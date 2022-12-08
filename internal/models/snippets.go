package models

import (
	"database/sql"
	"errors"
	"time"
)

type Snippet struct {
	ID      int
	Title   string
	Content string
	Created time.Time
	Expires time.Time
}

type SnippetModel struct {
	DB *sql.DB // connection pool
}

func (s *SnippetModel) ExampleTransaction() error {
	// always either call rollback() or commit() before function returns
	// or the connection will stay opened and not be returned to the connection pool
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// first query
	_, err = tx.Exec(``)
	if err != nil {
		return err
	}
	// second query
	_, err = tx.Exec(``)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SnippetModel) Insert(title string, content string, expires int) (int, error) {
	statement := `INSERT INTO snippetbox.snippets (
					 title,
					 content,
					 created,
					 expires
					 ) VALUES (
					   ?,
					   ?,
					   UTC_TIMESTAMP(),
					   DATE_ADD(UTC_TIMESTAMP(), INTERVAL ? DAY )
				   )`
	result, err := s.DB.Exec(statement, title, content, expires)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()

	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (s *SnippetModel) Get(id int) (*Snippet, error) {
	statement := `SELECT 
        id,
        title,
        content,
        created,
        expires
        FROM snippetbox.snippets WHERE expires > UTC_TIMESTAMP() AND id = ?`

	row := s.DB.QueryRow(statement, id)
	snippet := &Snippet{}
	err := row.Scan(&snippet.ID, &snippet.Title, &snippet.Content, &snippet.Created, &snippet.Expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}

	return snippet, nil
}

func (s *SnippetModel) Latest() ([]*Snippet, error) {
	statement := `
	SELECT
    	id,
    	title,
    	content,
    	created,
    	expires
    from snippetbox.snippets
	WHERE expires > UTC_TIMESTAMP() 
	ORDER BY id
	 DESC LIMIT 10
    `

	rows, err := s.DB.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // this will close the connection
	var snippets []*Snippet
	for rows.Next() {
		s := &Snippet{}
		err = rows.Scan(&s.ID, &s.Title, &s.Content, &s.Created, &s.Expires)
		if err != nil {
			return nil, err
		}
		snippets = append(snippets, s)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return snippets, nil
}
