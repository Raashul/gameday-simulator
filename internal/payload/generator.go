package payload

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"gameday-sim/internal/config"
)

// Generator handles payload generation
type Generator struct {
	config         *config.Config
	payloadData    *config.PayloadData
	rng            *rand.Rand
	currentRow     int
	currentCol     int
	direction      int     // 1 for right, -1 for left
	maxColInRow    int     // Track max column reached in current row
	polylineHeight float64 // Vertical extent of the base polyline
}

// NewGenerator creates a new payload generator
func NewGenerator(cfg *config.Config, payloadData *config.PayloadData) *Generator {
	// Calculate polyline vertical extent
	polylineHeight := calculatePolylineHeight(payloadData.BasePolyline.Coordinates)

	log.Printf("Generator initialized: polylineHeight=%.17f, delta.Lat=%.17f, rowSpacing=%.17f",
		polylineHeight, payloadData.Delta.Latitude, polylineHeight+payloadData.Delta.Latitude)

	return &Generator{
		config:         cfg,
		payloadData:    payloadData,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
		currentRow:     0,
		currentCol:     0,
		direction:      1, // Start moving right
		maxColInRow:    -1,
		polylineHeight: polylineHeight,
	}
}

// calculatePolylineHeight returns the vertical extent (max lat - min lat)
func calculatePolylineHeight(coords [][]float64) float64 {
	if len(coords) == 0 {
		return 0
	}

	minLat := coords[0][1]
	maxLat := coords[0][1]

	for _, coord := range coords {
		lat := coord[1]
		if lat < minLat {
			minLat = lat
		}
		if lat > maxLat {
			maxLat = lat
		}
	}

	return maxLat - minLat
}

// GenerateAll pre-generates all payloads for the simulation
func (g *Generator) GenerateAll() []OrderPayload {
	totalOrders := g.config.Simulation.TotalOrders
	activatedCount := g.config.Simulation.ActivatedCount

	payloads := make([]OrderPayload, 0, totalOrders)

	// Generate payloads for "activate" type orders
	for i := 0; i < activatedCount; i++ {
		payloads = append(payloads, g.generatePayload(i, TypeActivate))
	}

	// Generate payloads for "accepted" type orders
	for i := activatedCount; i < totalOrders; i++ {
		payloads = append(payloads, g.generatePayload(i, TypeAccepted))
	}

	// Shuffle payloads for random distribution
	//g.shuffle(payloads)

	return payloads
}

// generatePayload creates a single order payload
func (g *Generator) generatePayload(index int, orderType OrderType) OrderPayload {
	// Generate unique order number with zero-padded index
	orderNumber := fmt.Sprintf("%s%06d", g.config.Payload.OrderNumberPrefix, index+1)

	// Copy custom fields to avoid mutation
	customFields := make(map[string]interface{})
	for k, v := range g.config.Payload.CustomFields {
		customFields[k] = v
	}

	log.Printf("Generating payload %d (order: %s, type: %s) - Position: row=%d, col=%d, direction=%d",
		index, orderNumber, orderType, g.currentRow, g.currentCol, g.direction)

	// Generate GeoJSON polyline with offset
	geometry := g.generatePolyline(index)

	// Format coordinates for easy copy-paste to GeoJSON
	fmt.Printf("  \"coordinates\": [\n")
	for i, coord := range geometry.Coordinates {
		if i < len(geometry.Coordinates)-1 {
			fmt.Printf("    [%.17f, %.17f],\n", coord[0], coord[1])
		} else {
			fmt.Printf("    [%.17f, %.17f]\n", coord[0], coord[1])
		}
	}
	fmt.Printf("  ]\n")

	return OrderPayload{
		OrderNumber:  orderNumber,
		Location:     g.config.Payload.Location,
		POCOrder:     g.config.Payload.POCOrder,
		Timestamp:    time.Now(),
		Type:         orderType,
		CustomFields: customFields,
		Geometry:     geometry,
	}
}

// generatePolyline creates a GeoJSON LineString with offset based on zigzag pattern
func (g *Generator) generatePolyline(index int) *GeoJSONGeometry {
	baseCoords := g.payloadData.BasePolyline.Coordinates

	// Try to place the polyline, adjusting position if needed
	for {
		// Calculate offset based on current position
		lngOffset := g.payloadData.Delta.Longitude * float64(g.currentCol)

		// Row offset includes the full polyline height + delta spacing
		// This ensures rows are stacked like stairs, not overlapping
		rowSpacing := g.polylineHeight + g.payloadData.Delta.Latitude
		latOffset := -rowSpacing * float64(g.currentRow) // Negative to move down (south)

		// Create candidate coordinates
		coordinates := make([][]float64, len(baseCoords))
		for i, coord := range baseCoords {
			coordinates[i] = []float64{
				coord[0] + lngOffset, // longitude
				coord[1] + latOffset, // latitude
			}
		}

		// Check if all points are within boundary
		if g.isPolylineInBoundary(coordinates) {
			// Valid position, track max column and advance
			if g.currentCol > g.maxColInRow {
				g.maxColInRow = g.currentCol
			}

			// Advance based on direction
			if g.direction == 1 {
				g.currentCol++ // Going right, increment
			} else {
				g.currentCol-- // Going left, decrement
			}

			return &GeoJSONGeometry{
				Type:        "LineString",
				Coordinates: coordinates,
			}
		}

		// Doesn't fit, move to next row and flip direction
		g.currentRow++
		g.direction *= -1

		// Set starting column based on new direction
		if g.direction == 1 {
			// Going right, start from column 0
			g.currentCol = 0
		} else {
			// Going left, start from max column reached in previous row
			g.currentCol = g.maxColInRow
		}
		g.maxColInRow = -1 // Reset max for new row
	}
}

// isPolylineInBoundary checks if all points of a polyline are within the boundary polygon
func (g *Generator) isPolylineInBoundary(polyline [][]float64) bool {
	boundary := g.payloadData.Boundary.Coordinates

	// If no boundary is configured, allow all positions
	if len(boundary) == 0 {
		return true
	}

	// Use the first ring (exterior ring) of the polygon
	exteriorRing := boundary[0]

	// Check each point of the polyline
	for _, point := range polyline {
		if !g.isPointInPolygon(point[0], point[1], exteriorRing) {
			return false
		}
	}

	return true
}

// isPointInPolygon uses ray casting algorithm to check if point is inside polygon
func (g *Generator) isPointInPolygon(lng, lat float64, polygon [][]float64) bool {
	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		xi, yi := polygon[i][0], polygon[i][1]
		xj, yj := polygon[j][0], polygon[j][1]

		intersect := ((yi > lat) != (yj > lat)) &&
			(lng < (xj-xi)*(lat-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}

		j = i
	}

	return inside
}

// shuffle randomizes the order of payloads
func (g *Generator) shuffle(payloads []OrderPayload) {
	g.rng.Shuffle(len(payloads), func(i, j int) {
		payloads[i], payloads[j] = payloads[j], payloads[i]
	})
}

// DumpGeoJSON outputs all payloads and boundary as a GeoJSON FeatureCollection
func (g *Generator) DumpGeoJSON(payloads []OrderPayload) {
	features := []map[string]interface{}{}

	// Add boundary polygon as first feature
	if len(g.payloadData.Boundary.Coordinates) > 0 {
		boundaryFeature := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"name":         "Boundary",
				"stroke":       "#ff0000",
				"stroke-width": 2,
				"fill":         "#ff0000",
				"fill-opacity": 0.1,
			},
			"geometry": map[string]interface{}{
				"type":        "Polygon",
				"coordinates": g.payloadData.Boundary.Coordinates,
			},
		}
		features = append(features, boundaryFeature)
	}

	// Add base polyline for reference
	baseFeature := map[string]interface{}{
		"type": "Feature",
		"properties": map[string]interface{}{
			"name":         "Base Polyline (Row 0, Col 0)",
			"stroke":       "#0000ff",
			"stroke-width": 3,
		},
		"geometry": map[string]interface{}{
			"type":        "LineString",
			"coordinates": g.payloadData.BasePolyline.Coordinates,
		},
	}
	features = append(features, baseFeature)

	// Add all generated polylines
	for i, payload := range payloads {
		if payload.Geometry != nil {
			feature := map[string]interface{}{
				"type": "Feature",
				"properties": map[string]interface{}{
					"orderNumber":  payload.OrderNumber,
					"index":        i,
					"type":         string(payload.Type),
					"stroke":       getColorForType(payload.Type),
					"stroke-width": 2,
				},
				"geometry": map[string]interface{}{
					"type":        payload.Geometry.Type,
					"coordinates": payload.Geometry.Coordinates,
				},
			}
			features = append(features, feature)
		}
	}

	featureCollection := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
	}

	jsonData, err := json.MarshalIndent(featureCollection, "", "  ")
	if err != nil {
		log.Printf("Error marshaling GeoJSON: %v", err)
		return
	}

	// Create logs/geojsons directory
	logsDir := "logs/geojsons"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Printf("Error creating directory %s: %v", logsDir, err)
		return
	}

	// Create filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(logsDir, fmt.Sprintf("payloads_%s.json", timestamp))

	// Write to file
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		log.Printf("Error writing file %s: %v", filename, err)
		return
	}

	log.Printf("✓ GeoJSON saved to: %s", filename)
	log.Printf("  View at: https://geojson.io (drag and drop the file)")
}

// getColorForType returns a color based on order type
func getColorForType(orderType OrderType) string {
	if orderType == TypeActivate {
		return "#00ff00" // Green for activate
	}
	return "#ffaa00" // Orange for accepted
}
