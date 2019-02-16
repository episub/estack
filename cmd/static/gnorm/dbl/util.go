package dbl

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx"
)

func paginateCompare(fields []string, parentKey string, query string, index int) (string, string) {
	var compLeft []string
	var compRight []string

	for _, f := range fields {
		compLeft = append(compLeft, f)
		compRight = append(compRight, fmt.Sprintf("(SELECT %s FROM (%s) AS sd WHERE sd.%s = $%d)", f, query, parentKey, index))
	}

	return strings.Join(compLeft, ", "), strings.Join(compRight, ", ")
}

func rollbackErr(err error, tx *pgx.Tx) error {
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil
}
