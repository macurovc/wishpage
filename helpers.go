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
// Example: parseIDFromPath("/api/items/", "/api/items/123") returns 123, nil
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

// requestBodyData is a helper struct for extracting common fields from request bodies
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

	data := &requestBodyData{}
	ct := r.Header.Get("Content-Type")

	if strings.Contains(ct, "application/json") {
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		if v, ok := payload["name"].(string); ok {
			data.Name = v
		}
		if v, ok := payload["link"].(string); ok {
			data.Link = v
		}
		if v, ok := payload["family_member_id"].(string); ok {
			data.FamilyMemberID = v
		} else if v2, ok2 := payload["family_member_id"].(float64); ok2 {
			data.FamilyMemberID = strconv.Itoa(int(v2))
		}
		if v, ok := payload["price"].(string); ok {
			data.Price = v
		} else if v2, ok2 := payload["price"].(float64); ok2 {
			data.Price = strconv.FormatFloat(v2, 'f', -1, 64)
		}
	} else {
		// Default to form parsing (handles both form-urlencoded and default)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		data.Name = values.Get("name")
		data.Link = values.Get("link")
		data.Price = values.Get("price")
		data.FamilyMemberID = values.Get("family_member_id")
	}

	return data, nil
}
