package workflow

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
)

const (
	xmlPrefix = ""
	xmlIndent = "  "
)

type ProcessingConfig struct {
	XMLName xml.Name `xml:"processingMCP"`
	Choices Choices  `xml:"preconfiguredChoices>preconfiguredChoice"`
}

type Choices []Choice

// MarshalXML encodes the comment of each preconfiguredChoice.
//
// For example:
//
//	<!-- Store DIP -->
//	<preconfiguredChoice>
//	  <appliesTo>5e58066d-e113-4383-b20b-f301ed4d751c</appliesTo>
//	  <goToChain>8d29eb3d-a8a8-4347-806e-3d8227ed44a1</goToChain>
//	</preconfiguredChoice>
func (c Choices) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	indent := xml.CharData(fmt.Sprintf("\n%s%s", xmlIndent, xmlIndent))
	for _, item := range c {
		if err := e.EncodeToken(indent); err != nil {
			return err
		}
		if err := e.EncodeToken(xml.Comment(fmt.Sprintf(" %s ", item.Comment))); err != nil {
			return err
		}
		if err := e.Encode(item); err != nil {
			return err
		}
	}
	return e.Flush()
}

type Choice struct {
	XMLName   xml.Name `xml:"preconfiguredChoice"`
	Comment   string   `xml:"-"`
	AppliesTo string   `xml:"appliesTo"` // UUID.
	GoToChain string   `xml:"goToChain"` // UUID or URI.
}

func (c Choice) LinkID() uuid.UUID {
	if id, err := uuid.Parse(c.AppliesTo); err == nil {
		return id
	}
	return uuid.Nil
}

func (c Choice) ChainID() uuid.UUID {
	if id, err := uuid.Parse(c.GoToChain); err == nil {
		return id
	}
	return uuid.Nil
}

func (c Choice) Value() string {
	return c.GoToChain
}

func ParseConfigFile(path string) ([]Choice, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseConfig(bytes.NewReader(blob))
}

func ParseConfig(reader io.Reader) ([]Choice, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var config ProcessingConfig
	err = xml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}

	return config.Choices, nil
}

func SaveConfigFile(path string, choices []Choice) error {
	config := ProcessingConfig{
		Choices: choices,
	}

	blob, err := xml.MarshalIndent(config, xmlPrefix, xmlIndent)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(blob)

	return err
}
