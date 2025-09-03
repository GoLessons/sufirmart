package order

import (
	"sufirmart/internal/domain"
)

type Processor struct {
}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) ProcessOrder(number domain.OrderNumber) {

}
