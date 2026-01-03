package payload

import (
	"fmt"
	"math/rand"
	"time"

	"gameday-sim/internal/config"
)

// Generator handles payload generation
type Generator struct {
	config *config.Config
	rng    *rand.Rand
}

// NewGenerator creates a new payload generator
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		config: cfg,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
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
	g.shuffle(payloads)

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

	return OrderPayload{
		OrderNumber:  orderNumber,
		Location:     g.config.Payload.Location,
		POCOrder:     g.config.Payload.POCOrder,
		Timestamp:    time.Now(),
		Type:         orderType,
		CustomFields: customFields,
	}
}

// shuffle randomizes the order of payloads
func (g *Generator) shuffle(payloads []OrderPayload) {
	g.rng.Shuffle(len(payloads), func(i, j int) {
		payloads[i], payloads[j] = payloads[j], payloads[i]
	})
}
