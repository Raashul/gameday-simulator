package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// PayloadData represents the baseline payload configuration from JSON
type PayloadData struct {
	BasePolyline BasePolyline    `json:"basePolyline"`
	Boundary     PolygonBoundary `json:"boundary"`
	Delta        CoordinateDelta `json:"delta"`
}

// GeoJSONLineString represents a GeoJSON LineString
type GeoJSONLineString struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

// GeoJSONPolygon represents a GeoJSON Polygon
type GeoJSONPolygon struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

// PayloadDataJSON is the JSON structure from payload.json
type PayloadDataJSON struct {
	BasePolyline GeoJSONLineString `json:"basePolyline"`
	Boundary     GeoJSONPolygon    `json:"boundary"`
	Delta        CoordinateDelta   `json:"delta"`
}

// LoadPayloadData loads the payload configuration from JSON file
func LoadPayloadData(filePath string) (*PayloadData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read payload file: %w", err)
	}

	var jsonData PayloadDataJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse payload JSON: %w", err)
	}

	// Convert to internal format
	payloadData := &PayloadData{
		BasePolyline: BasePolyline{
			Coordinates: jsonData.BasePolyline.Coordinates,
		},
		Boundary: PolygonBoundary{
			Coordinates: jsonData.Boundary.Coordinates,
		},
		Delta: jsonData.Delta,
	}

	return payloadData, nil
}
