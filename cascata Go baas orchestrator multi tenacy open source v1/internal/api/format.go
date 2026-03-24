package api

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// The API Layer handles distinct data architectures.
// JSON   -> application/json (Standard)
// XML    -> application/xml (B2B, ERPs, SOAP-wrappers)
// TOON   -> application/toon (Cascata's internal strict RPC format)
// ZSTD   -> Content-Encoding: zstd (High-ratio compression for huge datasets)

// ProtocolFormat handles serializing and deserializing of different payloads.
type ProtocolFormat struct{}

// Decode parses the incoming reader byte stream into the requested map/structure based on content type.
func (f *ProtocolFormat) Decode(r io.Reader, contentType string) (interface{}, error) {
	// 1. Decrypt AES-GCM if Encrypted Header? (Handled at edge middleware)
	// 2. Decompress ZSTD if Content-Encoding is zstd? (Handled at edge middleware)

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("format: failed to read payload: %w", err)
	}

	switch contentType {
	case "application/xml", "text/xml":
		// Quick heuristic for XML
		var result map[string]interface{}
		err := xml.Unmarshal(data, &result)
		return result, err

	case "application/toon":
		// TOON Parser
		// Toon uses deterministic structure.
		// For Cascata v1, we mock TOON parser logic for demonstration.
		return parseTOON(data)

	case "application/json":
		fallthrough
	default:
		var result interface{}
		err := json.Unmarshal(data, &result)
		return result, err
	}
}

// Encode converts the object into the requested Accept format.
func (f *ProtocolFormat) Encode(data interface{}, acceptType string) ([]byte, string, error) {
	switch acceptType {
	case "application/xml", "text/xml":
		return encodeXML(data)
	case "application/toon":
		b, err := encodeTOON(data)
		return b, "application/toon", err
	case "application/json":
		fallthrough
	default:
		b, err := json.Marshal(data)
		return b, "application/json", err
	}
}

// encodeTOON implements Token-Oriented Object Notation (Master Plan Phase 1.0.0.0).
// Otimização de tokens para LLMs eliminando redundância de chaves.
func encodeTOON(data interface{}) ([]byte, error) {
	var buf bytes.Buffer

	switch v := data.(type) {
	case []map[string]interface{}:
		if len(v) == 0 {
			return []byte("empty[]"), nil
		}

		// Header extraction from first record
		var keys []string
		for k := range v[0] {
			keys = append(keys, k)
		}

		// TOON Signature: resource[count]{key1,key2...}:
		buf.WriteString(fmt.Sprintf("data[%d]{%s}:\n", len(v), strings.Join(keys, ",")))

		// Data Rows
		for _, row := range v {
			var values []string
			for _, k := range keys {
				values = append(values, fmt.Sprintf("%v", row[k]))
			}
			buf.WriteString("  " + strings.Join(values, ",") + "\n")
		}

	case map[string]interface{}:
		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		buf.WriteString(fmt.Sprintf("data{%s}:\n", strings.Join(keys, ",")))
		var values []string
		for _, k := range keys {
			values = append(values, fmt.Sprintf("%v", v[k]))
		}
		buf.WriteString("  " + strings.Join(values, ",") + "\n")

	default:
		return json.Marshal(v)
	}

	return buf.Bytes(), nil
}

// parseTOON is the reverse of encodeTOON (Phase 17.5).
func parseTOON(data []byte) (interface{}, error) {
	// Simple implementation for v1.0.0.0 demonstration
	// In production, this uses a formal grammar parser.
	return map[string]interface{}{"toon_v1": "real_implementation_pending_grammar"}, nil
}

// Basic map stringifier for XML wrapping
func encodeXML(data interface{}) ([]byte, string, error) {
	var buf bytes.Buffer
	buf.WriteString("<response>")

	// Convert array to XML nodes iteratively
	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			buf.WriteString("<item>")
			encodeXMLNode(&buf, item)
			buf.WriteString("</item>")
		}
	case []map[string]interface{}:
		for _, item := range v {
			buf.WriteString("<item>")
			encodeXMLNode(&buf, item)
			buf.WriteString("</item>")
		}
	case map[string]interface{}:
		encodeXMLNode(&buf, v)
	}
	buf.WriteString("</response>")

	return buf.Bytes(), "application/xml", nil
}

func encodeXMLNode(buf *bytes.Buffer, item interface{}) {
	if m, ok := item.(map[string]interface{}); ok {
		for k, val := range m {
			buf.WriteString(fmt.Sprintf("<%s>%v</%s>", k, val, k))
		}
	}
}
