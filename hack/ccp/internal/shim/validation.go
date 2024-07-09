package shim

import (
	"encoding/csv"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/artefactual/archivematica/hack/ccp/internal/shim/gen"
)

type validationError struct{} //nolint: unused

func (err *validationError) Error() string { //nolint: unused
	return "validation error"
}

var validators = map[gen.V2BetaValidateValidator]validator{
	gen.V2BetaValidateValidatorAvalon: avalonValidator{},
	gen.V2BetaValidateValidatorRights: rightsValidator{},
}

var acceptedValidators = strings.Join([]string{
	string(gen.V2BetaValidateValidatorAvalon),
	string(gen.V2BetaValidateValidatorRights),
}, ", ")

func validateContentType(req *http.Request) error {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		return fmt.Errorf("header Content-Type is missing")
	}

	mimeType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("invalid Content-Type header: %v", err)
	}

	if mimeType != "text/csv" || params["charset"] != "utf-8" {
		return fmt.Errorf("content type should be \"text/csv; charset=utf-8\"")
	}

	return nil
}

func loadValidator(name gen.V2BetaValidateValidator) (validator, error) {
	validator, ok := validators[name]
	if !ok {
		return nil, fmt.Errorf("unknown validator, accepted values: %s", acceptedValidators)
	}

	return validator, nil
}

type validator interface {
	validate(r io.Reader) error
}

type avalonValidator struct{}

var _ validator = (*avalonValidator)(nil)

func (v avalonValidator) validate(r io.Reader) error {
	cr := csv.NewReader(r)
	cr.Comma = ','

	if adminData, err := cr.Read(); err != nil {
		return err
	} else if err := v.checkAdminData(adminData); err != nil {
		return err
	}

	var fileCols, opCols []int
	if headerData, err := cr.Read(); err != nil {
		return err
	} else if err := v.checkHeaderData(headerData); err != nil {
		return err
	} else {
		fileCols = v.fileColumns(headerData)
		opCols = v.opColumns(headerData)
		if err := v.checkFieldPairs(headerData); err != nil {
			return err
		}
	}

	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := v.checkFileExts(row, fileCols); err != nil {
			return err
		}
		if err := v.checkOpFields(row, opCols); err != nil {
			return err
		}
	}

	return nil
}

func (v avalonValidator) checkAdminData(row []string) error {
	if len(row) < 2 || row[0] == "" || row[1] == "" {
		return fmt.Errorf("administrative data must include reference name and author")
	}
	return nil
}

func (v avalonValidator) checkHeaderData(row []string) error {
	allHeaders := []string{
		"Bibliographic ID", "Bibliographic ID Label", "Other Identifier",
		"Other Identifier Type", "Title", "Creator", "Contributor", "Genre",
		"Publisher", "Date Created", "Date Issued", "Abstract", "Language",
		"Physical Description", "Related Item URL", "Related Item Label",
		"Topical Subject", "Geographic Subject", "Temporal Subject",
		"Terms of Use", "Table of Contents", "Statement of Responsibility",
		"Note", "Note Type", "Publish", "Hidden", "File", "Label", "Offset",
		"Skip Transcoding", "Absolute Location", "Date Ingested",
	}
	reqHeaders := []string{"Title", "Date Issued", "File"}
	uniqueHeaders := []string{
		"Bibliographic ID", "Bibliographic ID Label", "Title", "Date Created",
		"Date Issued", "Abstract", "Physical Description", "Terms of Use",
	}

	headerSet := make(map[string]int)
	for _, header := range row {
		headerSet[strings.TrimSpace(header)]++
	}

	for _, header := range row {
		if strings.TrimSpace(header) != header {
			return fmt.Errorf("header fields cannot have leading or trailing blanks. Invalid field: %s", header)
		}
	}

	for _, header := range row {
		found := false
		for _, validHeader := range allHeaders {
			if header == validHeader {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("manifest includes invalid metadata field: %s", header)
		}
	}

	for _, uniqueHeader := range uniqueHeaders {
		if headerSet[uniqueHeader] > 1 {
			return fmt.Errorf("a non-repeatable header field is repeated: %s", uniqueHeader)
		}
	}

	for _, reqHeader := range reqHeaders {
		if headerSet[reqHeader] == 0 && headerSet["Bibliographic ID"] == 0 {
			return fmt.Errorf("one of the required headers is missing: Title, Date Issued, File")
		}
	}

	return nil
}

func (v avalonValidator) fileColumns(row []string) []int {
	var columns []int
	for i, field := range row {
		if field == "File" {
			columns = append(columns, i)
		}
	}
	return columns
}

func (v avalonValidator) opColumns(row []string) []int {
	var columns []int
	for i, field := range row {
		if field == "Publish" || field == "Hidden" {
			columns = append(columns, i)
		}
	}
	return columns
}

func (v avalonValidator) checkFieldPairs(row []string) error {
	fieldSet := make(map[string]bool)
	for _, field := range row {
		fieldSet[field] = true
	}

	pairs := [][]string{
		{"Other Identifier", "Other Identifier Type"},
		{"Related Item URL", "Related Item Label"},
		{"Note", "Note Type"},
	}

	for _, pair := range pairs {
		if fieldSet[pair[0]] != fieldSet[pair[1]] {
			return fmt.Errorf("%s field missing its required pair", pair[0])
		}
	}

	return nil
}

func (v avalonValidator) checkFileExts(row []string, fileCols []int) error {
	for _, c := range fileCols {
		if c >= len(row) {
			continue
		}
		filepath := row[c]
		periods := strings.Count(filepath, ".")
		if periods > 1 && !strings.Contains(filepath, ".high.") &&
			!strings.Contains(filepath, ".medium.") &&
			!strings.Contains(filepath, ".low.") {
			return fmt.Errorf("filepath %s contains more than one period", filepath)
		}
	}
	return nil
}

func (v avalonValidator) checkOpFields(row []string, opCols []int) error {
	for _, c := range opCols {
		if c >= len(row) {
			continue
		}
		value := strings.ToLower(row[c])
		if value != "" && value != "yes" && value != "no" {
			return fmt.Errorf("publish/hidden fields must have boolean value (yes or no). Value is %s", row[c])
		}
	}
	return nil
}

type rightsValidator struct{}

func (v rightsValidator) validate(r io.Reader) error {
	return nil
}
