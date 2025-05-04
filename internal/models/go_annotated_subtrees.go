package models

import (
	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"log"
	"mobetta/internal/db"
)

type GoAnnotatedSubtreesModel struct {
	db *db.PostgresDb
}

func NewAnnotatedAstModel(db *db.PostgresDb) *GoAnnotatedSubtreesModel {
	return &GoAnnotatedSubtreesModel{db: db}
}

func (model *GoAnnotatedSubtreesModel) Create() error {
	_, err := model.db.Exec(`CREATE TABLE IF NOT EXISTS go_annotated_subtrees  (
									id bigserial PRIMARY KEY,
									source_file text,
									start_line_number int,
									end_line_number int,
									inorder_node_string text,
									raw_subtree_string text,
									embedding vector(219))`)
	return err
}

func (model *GoAnnotatedSubtreesModel) Drop() error {
	_, err := model.db.Exec(`DROP TABLE IF EXISTS go_annotated_subtrees;`)
	return err
}

func (model *GoAnnotatedSubtreesModel) Insert(sourceFile string, startLineNumber uint, endLineNumber uint, inorderNodeString string, rawSubtreeString string, embedding []float32) error {
	_, err := model.db.Exec("INSERT INTO go_annotated_subtrees (source_file, start_line_number, end_line_number, inorder_node_string, raw_subtree_string, embedding) VALUES ($1,$2,$3,$4,$5,$6)",
		sourceFile, startLineNumber, endLineNumber, inorderNodeString, rawSubtreeString, pgvector.NewVector(embedding))
	return err
}

func (model *GoAnnotatedSubtreesModel) FetchSimilar(embedding []float32, limit int) ([]string, error) {
	var fetchedStrings []string
	rows, err := model.db.Query("SELECT raw_subtree_string FROM go_annotated_subtrees ORDER BY embedding <-> $1 LIMIT $2", pgvector.NewVector(embedding), limit)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {
		var nodeString string
		err := rows.Scan(&nodeString)
		if err != nil {
			log.Fatal(err)
		}
		fetchedStrings = append(fetchedStrings, nodeString)
	}

	return fetchedStrings, nil
}
