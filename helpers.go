package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// parseIDFromPath extracts an integer ID from a URL path after the given prefix.
// Example: parseIDFromPath("/api/items/", "/api/items/123") returns 123, nil.
func parseIDFromPath(prefix, path string) (int, error) {
	seg, ok := strings.CutPrefix(path, prefix)
	if !ok {
		return 0, http.ErrNotSupported
	}
	idStr, _, _ := strings.Cut(seg, "/")
	return strconv.Atoi(idStr)
}

// parseRequestBody parses form or JSON request body and returns a map of string values.
// Handles both application/x-www-form-urlencoded and application/json content types.
func parseRequestBody(r *http.Request) (map[string]string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	ct := r.Header.Get("Content-Type")

	if strings.Contains(ct, "application/json") {
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		// Convert all values to strings
		for k, v := range payload {
			switch val := v.(type) {
			case string:
				result[k] = val
			case float64:
				result[k] = strconv.FormatFloat(val, 'f', -1, 64)
			case int:
				result[k] = strconv.Itoa(val)
			case bool:
				result[k] = strconv.FormatBool(val)
			}
		}
	} else {
		// Default to form parsing
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		for k := range values {
			result[k] = values.Get(k)
		}
	}

	return result, nil
}

// requestBodyData is a helper struct for extracting common fields from request bodies.
type requestBodyData struct {
	Name           string
	Link           string
	Price          string
	FamilyMemberID string
}

// parseItemRequestBody parses form or JSON request body and extracts item-related fields.
// This is a specialized version of parseRequestBody for item creation/update operations.
func parseItemRequestBody(r *http.Request) (*requestBodyData, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	ct := r.Header.Get("Content-Type")

	if strings.Contains(ct, "application/json") {
		return parseJSONRequestBody(body)
	}

	return parseFormRequestBody(body)
}

// parseJSONRequestBody extracts request body data from JSON payload.
func parseJSONRequestBody(body []byte) (*requestBodyData, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	data := &requestBodyData{
		Name:           extractString(payload, "name"),
		Link:           extractString(payload, "link"),
		FamilyMemberID: extractStringOrNumber(payload, "family_member_id"),
		Price:          extractStringOrNumber(payload, "price"),
	}

	return data, nil
}

// parseFormRequestBody extracts request body data from form-urlencoded payload.
func parseFormRequestBody(body []byte) (*requestBodyData, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}

	data := &requestBodyData{
		Name:           values.Get("name"),
		Link:           values.Get("link"),
		Price:          values.Get("price"),
		FamilyMemberID: values.Get("family_member_id"),
	}

	return data, nil
}

// extractString extracts a string value from a map payload.
func extractString(payload map[string]any, key string) string {
	if v, ok := payload[key].(string); ok {
		return v
	}
	return ""
}

// extractStringOrNumber extracts a value that can be either a string or number from a map payload.
func extractStringOrNumber(payload map[string]any, key string) string {
	if v, ok := payload[key].(string); ok {
		return v
	}
	if v, ok := payload[key].(float64); ok {
		if key == "price" {
			return strconv.FormatFloat(v, 'f', -1, 64)
		}
		return strconv.Itoa(int(v))
	}
	return ""
}
