package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"mobetta/internal/db"
	"mobetta/internal/models"
	"mobetta/internal/util"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tree-sitter/go-tree-sitter"
	treesittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

var _language *tree_sitter.Language

func language() *tree_sitter.Language {
	if _language == nil {
		_language = tree_sitter.NewLanguage(treesittergo.Language())
	}
	return _language
}

func isMeaningfulDeclaration(node *tree_sitter.Node) bool {
	return node.Kind() == "function_declaration" || node.Kind() == "method_declaration" || node.Kind() == "type_declaration"
}

func getNodeFreq(s string) map[string]int {
	nodeFreqs := make(map[string]int)
	for _, node := range strings.Split(s, ":") {
		nodeFreqs[node]++
	}
	return nodeFreqs
}

func cosineSimilarity(vec1, vec2 map[string]int) float64 {
	dotProduct := 0.0
	normVec1 := 0.0
	normVec2 := 0.0

	// Calculate dot product and norm of vec1
	for key, val := range vec1 {
		dotProduct += float64(val * vec2[key])
		normVec1 += math.Pow(float64(val), 2)
	}
	normVec1 = math.Sqrt(normVec1)

	// Calculate norm of vec2
	for _, val := range vec2 {
		normVec2 += math.Pow(float64(val), 2)
	}
	normVec2 = math.Sqrt(normVec2)

	if normVec1 == 0 || normVec2 == 0 {
		return 0.0
	}

	return dotProduct / (normVec1 * normVec2)
}

func printSimilarityTable(subtreeNodeFreq map[uintptr]map[string]int) {
	for k1, v1 := range subtreeNodeFreq {
		for k2, v2 := range subtreeNodeFreq {
			fmt.Printf("%d:%d\t%f\n", k1, k2, cosineSimilarity(v1, v2))
		}
	}
}

func calculateSubtreeStrings(node *tree_sitter.Node, source []byte, subtreeStrings map[uintptr]string) string {
	// nodeText := node.Utf8Text(source)
	nodeKind := node.Kind()
	subtreeSlice := []string{nodeKind}
	if nodeKind == "type_identifier" {
		subtreeSlice = []string{"type_identifier_" + node.Utf8Text(source)}
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child.IsNamed() {
			if ss, ok := subtreeStrings[child.Id()]; ok {
				subtreeSlice = append(subtreeSlice, ss)
			}
		}
	}
	return strings.Join(subtreeSlice, ":")
}

func ingestSubtrees(roots []*tree_sitter.Node, annotationsModel *models.GoAnnotatedSubtreesModel, sourceFileName string, source []byte, subtreeStrings map[uintptr]string, subtreeNodeFreq map[uintptr]map[string]int, embeddingFn func(map[string]int) []float32) error {
	for _, node := range roots {
		err := annotationsModel.Insert(sourceFileName, node.StartPosition().Row, node.EndPosition().Row, subtreeStrings[node.Id()], node.Utf8Text(source), embeddingFn(subtreeNodeFreq[node.Id()]))
		if err != nil {
			return err
		}
	}
	return nil
}

func getClosestSubtree(node *tree_sitter.Node, annotationsModel *models.GoAnnotatedSubtreesModel, subtreeNodeFreq map[uintptr]map[string]int, embeddingFn func(map[string]int) []float32, limit int) ([]string, error) {
	embedding := embeddingFn(subtreeNodeFreq[node.Id()])
	return annotationsModel.FetchSimilar(embedding, limit)
}

func getClosestSubtrees(nodes []*tree_sitter.Node, annotationsModel *models.GoAnnotatedSubtreesModel, source []byte, subtreeNodeFreq map[uintptr]map[string]int, embeddingFn func(map[string]int) []float32, limit int) ([][]string, error) {
	results := make([][]string, len(nodes))
	for i, node := range nodes {
		subtreeString, err := getClosestSubtree(node, annotationsModel, subtreeNodeFreq, embeddingFn, limit)
		if err != nil {
			return nil, err
		}
		results[i] = []string{node.Utf8Text(source)}
		results[i] = append(results[i], subtreeString...)
	}
	return results, nil
}

func annotateAndGetSubtrees(node *tree_sitter.Node, source []byte, subtreeStrings map[uintptr]string, subtreeNodeFreq map[uintptr]map[string]int) []*tree_sitter.Node {
	var annotatedSubtreeNodes []*tree_sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		annotatedSubtreeNodes = append(annotatedSubtreeNodes, annotateAndGetSubtrees(node.Child(uint(i)), source, subtreeStrings, subtreeNodeFreq)...)
	}
	subtreeStrings[node.Id()] = calculateSubtreeStrings(node, source, subtreeStrings)
	if isMeaningfulDeclaration(node) {
		subtreeNodeFreq[node.Id()] = getNodeFreq(subtreeStrings[node.Id()])
		return append(annotatedSubtreeNodes, node)
	}
	return annotatedSubtreeNodes
}

func printAnnotatedTree(node *tree_sitter.Node, indent int, source []byte, subtreeStrings map[uintptr]string, subtreeNodeFreq map[uintptr]map[string]int) {
	if !node.IsNamed() {
		return
	}
	if isMeaningfulDeclaration(node) {
		subtreeString := "None"
		if f, ok := subtreeStrings[node.Id()]; ok {
			subtreeString = f
		}
		subtreeNodeFreqs := make(map[string]int)
		if f := subtreeNodeFreq[node.Id()]; f != nil {
			subtreeNodeFreqs = f
		}
		fmt.Printf("%s<%s>:\n%s\n%v\n", indentString(indent), node.Kind(), subtreeString, subtreeNodeFreqs)
	} else {
		fmt.Printf("%s<%s>\n", indentString(indent), node.Kind())
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		printAnnotatedTree(node.Child(uint(i)), indent+1, source, subtreeStrings, subtreeNodeFreq)
	}
}

func indentString(indent int) string {
	s := ""
	for i := 0; i < indent; i++ {
		s += "  "
	}
	return s
}

func fetchNodeTypes() []string {
	nodeTypes := make([]string, 0)
	file, err := os.Open("internal/data/node_types_go.txt")
	if err != nil {
		log.Fatalf("Error opening file: %s", err.Error())
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		nodeTypes = append(nodeTypes, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %s", err.Error())
	}
	return nodeTypes
}

func nodeFreqMapToFloatArray(nodeFreqMap map[string]int, nodeTypes []string) []float32 {
	floatArray := make([]float32, len(nodeTypes))
	for i, term := range nodeTypes {
		floatArray[i] = float32(nodeFreqMap[term])
	}
	return floatArray
}

type SourceMetadata struct {
	tree            *tree_sitter.Tree
	rootNode        *tree_sitter.Node
	subtreeStrings  map[uintptr]string
	subtreeNodeFreq map[uintptr]map[string]int
	subtreeRoots    []*tree_sitter.Node
	source          []byte
}

func (sm *SourceMetadata) Close() {
	sm.tree.Close()
}

func processSourceFile(sourceFile string, verbose bool) (*SourceMetadata, error) {
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		log.Fatal(err)
	}

	// Create a Tree-sitter parser
	parser := tree_sitter.NewParser()

	// Set the Go language grammar
	_ = parser.SetLanguage(language())

	// Parse the code
	tree := parser.Parse(source, nil)
	if tree == nil {
		return nil, fmt.Errorf("Error parsing code")
	}

	rootNode := tree.RootNode()
	subtreeStrings := make(map[uintptr]string)
	subtreeNodeFreq := make(map[uintptr]map[string]int)

	subtreeRoots := annotateAndGetSubtrees(rootNode, source, subtreeStrings, subtreeNodeFreq)

	if verbose {
		// Print the annotated tree
		printAnnotatedTree(rootNode, 0, source, subtreeStrings, subtreeNodeFreq)
		// Print similarity table for all functions in the ingested file
		printSimilarityTable(subtreeNodeFreq)
	}

	return &SourceMetadata{tree: tree, rootNode: rootNode, subtreeStrings: subtreeStrings, subtreeNodeFreq: subtreeNodeFreq, subtreeRoots: subtreeRoots, source: source}, nil
}

func findFilesWithExtension(baseDir, extension string) ([]string, error) {
	var fileList []string

	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, err)
			return nil // Continue walking despite the error
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), extension) {
			fileList = append(fileList, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %s: %w", baseDir, err)
	}

	return fileList, nil
}

func ingestRepo(repoLocation string, annotationsModel *models.GoAnnotatedSubtreesModel, embeddingFn func(map[string]int) []float32, verbose bool) error {
	files, err := findFilesWithExtension(repoLocation, ".go")
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		func(f string) {
			defer wg.Done()
			sourceMetadata, err := processSourceFile(file, verbose)
			defer sourceMetadata.Close()
			if err != nil {
				log.Fatal(err)
			}
			err = ingestSubtrees(sourceMetadata.subtreeRoots, annotationsModel, file, sourceMetadata.source, sourceMetadata.subtreeStrings, sourceMetadata.subtreeNodeFreq, embeddingFn)
			if err != nil {
				log.Fatal(err)
			}
		}(file)
	}

	wg.Wait()
	return nil
}

func main() {
	operationPtr := flag.String("operation", "", "values: ingest-file, ingest-repo, ingest-repos, search-by-embedding")
	sourceFilePtr := flag.String("sourceFile", "", "Relative path to the source file in the repo")
	repoLocationPtr := flag.String("repoLocation", "", "Absolute path to the source repo")
	repoURLPtr := flag.String("repoURL", "", "Github repo URL")
	verbosePtr := flag.Bool("verbose", false, "Verbose output")
	repoURLFilePtr := flag.String("repoURLFile", "", "Repo Location file")
	reposBaseDirPtr := flag.String("reposBaseDir", "", "Base directory for repos")
	numResultsPtr := flag.Int("numResults", 1, "Number of results to return")

	flag.Parse()

	sourceFile := *sourceFilePtr
	repoLocation := *repoLocationPtr
	repoURL := *repoURLPtr
	operation := *operationPtr
	verbose := *verbosePtr
	repoURLFile := *repoURLFilePtr
	reposBaseDir := *reposBaseDirPtr
	numResults := *numResultsPtr

	if operation == "" {
		flag.Usage()
		return
	}

	if operation == "ingest-file" {
		if sourceFile == "" || repoLocation == "" || repoURL == "" {
			flag.Usage()
			return
		}
	} else if operation == "ingest-repo" {
		if repoLocation == "" || repoURL == "" {
			flag.Usage()
			return
		}
	} else if operation == "ingest-repos" {
		if repoURLFile == "" {
			flag.Usage()
			return
		}
	} else if operation == "search-by-embedding" {
		if sourceFile == "" {
			flag.Usage()
			return
		}
	}

	pg := db.NewPostgresDb("host=localhost port=5432 user=postgres password=password dbname=postgres sslmode=disable")
	err := pg.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer pg.Close()

	annotatedAst := models.NewAnnotatedAstModel(pg)
	err = annotatedAst.Create()
	if err != nil {
		log.Fatal(err)
	}

	nodeTypes := fetchNodeTypes()
	embeddingFn := func(nodeFreqMap map[string]int) []float32 {
		return nodeFreqMapToFloatArray(nodeFreqMap, nodeTypes)
	}

	if operation == "ingest-file" {
		sourceMetadata, err := processSourceFile(sourceFile, verbose)
		defer sourceMetadata.Close()
		if err != nil {
			log.Fatal(err)
		}
		err = ingestSubtrees(sourceMetadata.subtreeRoots, annotatedAst, sourceFile, sourceMetadata.source, sourceMetadata.subtreeStrings, sourceMetadata.subtreeNodeFreq, embeddingFn)
		if err != nil {
			log.Fatal(err)
		}
	} else if operation == "ingest-repo" {
		err = ingestRepo(repoLocation, annotatedAst, embeddingFn, verbose)
		if err != nil {
			log.Fatal(err)
		}
	} else if operation == "ingest-repos" {
		repoURLs, err := util.GetRepoURLs(repoURLFile)
		if err != nil {
			log.Fatal(err)
		}
		var wg sync.WaitGroup
		barrier := make(chan struct{}, 16)
		for _, repoURL := range repoURLs {
			barrier <- struct{}{}
			wg.Add(1)
			repoName := util.GetRepoName(repoURL)
			repoOrg := util.GetRepoOrg(repoURL)
			fmt.Println("Ingesting repo: ", repoOrg, repoName)
			err = ingestRepo(fmt.Sprintf("%s/%s/%s", reposBaseDir, repoOrg, repoName), annotatedAst, embeddingFn, verbose)
			if err != nil {
				log.Print("Unable to ingest repo: ", repoOrg, repoName)
			}
			<-barrier
		}
		wg.Wait()
	} else if operation == "search-by-embedding" {
		var subtreeResults [][]string
		sourceMetadata, err := processSourceFile(sourceFile, verbose)
		if err != nil {
			log.Fatal(err)
		}
		subtreeResults, err = getClosestSubtrees(sourceMetadata.subtreeRoots, annotatedAst, sourceMetadata.source, sourceMetadata.subtreeNodeFreq, embeddingFn, numResults)
		if err != nil {
			log.Fatal(err)
		}
		for _, result := range subtreeResults {
			fmt.Println("Provided function:\n", strings.Join(result, "\n\n Embedding Match:\n"))
		}
	}
}
