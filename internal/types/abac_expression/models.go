package abac_expression

import (
	"fmt"

	accessGovernanceType "github.com/collibra/access-governance-go-sdk/types"

	"github.com/collibra/access-governance-terraform-provider/internal/utils"
)

type BinaryExpression struct {
	Literal         *bool            `json:"literal,omitempty"`
	Comparison      *AbacComparison  `json:"comparison,omitempty"`
	Aggregator      *Aggregator      `json:"aggregator,omitempty"`
	UnaryExpression *UnaryExpression `json:"unaryExpression,omitempty"`
}

func (b BinaryExpression) ToGqlInput() (*accessGovernanceType.AbacComparisonExpressionInput, error) {
	var comparison *accessGovernanceType.AbacComparisonExpressionComparisonInput
	var aggregator *accessGovernanceType.AbacComparisonExpressionAggregatorInput
	var unaryExpression *accessGovernanceType.AbacComparisonExpressionUnaryExpressionInput

	var err error

	if b.Comparison != nil {
		comparison = utils.Ptr(b.Comparison.ToGqlInput())
	} else if b.Aggregator != nil {
		aggregator, err = b.Aggregator.ToGqlInput()
		if err != nil {
			return nil, fmt.Errorf("aggregator to gql input: %w", err)
		}
	} else if b.UnaryExpression != nil {
		unaryExpression, err = b.UnaryExpression.ToGqlInput()
		if err != nil {
			return nil, fmt.Errorf("unaryExpression to gql input: %w", err)
		}
	}

	return &accessGovernanceType.AbacComparisonExpressionInput{
		Literal:         b.Literal,
		Comparison:      comparison,
		Aggregator:      aggregator,
		UnaryExpression: unaryExpression,
	}, nil
}

//go:generate go run github.com/raito-io/enumer -type=AbacOperator -values -gqlgen -yaml -json -trimprefix=AbacOperator
type AbacOperator int

const (
	AbacOperatorHasTag AbacOperator = iota
	AbacOperatorContainsTag
	AbacOperatorPropertyEquals
	AbacOperatorPropertyIn
)

type AbacComparison struct {
	Operator     AbacOperator `json:"operator"`
	LeftOperand  string       `json:"leftOperand"`
	RightOperand Operand      `json:"rightOperand"`
}

func (c AbacComparison) ToGqlInput() accessGovernanceType.AbacComparisonExpressionComparisonInput {
	return accessGovernanceType.AbacComparisonExpressionComparisonInput{
		Operator:     accessGovernanceType.AbacComparisonExpressionComparisonOperator(c.Operator.String()),
		LeftOperand:  c.LeftOperand,
		RightOperand: c.RightOperand.ToGqlInput(),
	}
}

type Operand struct {
	Literal *Literal `json:"literal,omitempty"`
}

func (o Operand) ToGqlInput() accessGovernanceType.AbacComparisonExpressionOperandInput {
	var literal *accessGovernanceType.AbacComparisonExpressionLiteral

	if o.Literal != nil {
		literal = utils.Ptr(o.Literal.ToGqlInput())
	}

	return accessGovernanceType.AbacComparisonExpressionOperandInput{
		Literal: literal,
	}
}

type Literal struct {
	Bool       *bool    `json:"bool,omitempty"`
	String     *string  `json:"string,omitempty"`
	StringList []string `json:"stringList,omitempty"`
}

func (l Literal) ToGqlInput() accessGovernanceType.AbacComparisonExpressionLiteral {
	return accessGovernanceType.AbacComparisonExpressionLiteral{
		Bool:       l.Bool,
		String:     l.String,
		StringList: l.StringList,
	}
}

//go:generate go run github.com/raito-io/enumer -type=AggregatorOperator -values -gqlgen -yaml -json -trimprefix=AggregatorOperator
type AggregatorOperator int

const (
	AggregatorOperatorAnd AggregatorOperator = iota
	AggregatorOperatorOr
)

type Aggregator struct {
	Operator AggregatorOperator `json:"operator"`
	Operands []BinaryExpression `json:"operands"`
}

func (a Aggregator) ToGqlInput() (*accessGovernanceType.AbacComparisonExpressionAggregatorInput, error) {
	operands := make([]accessGovernanceType.AbacComparisonExpressionInput, 0, len(a.Operands))

	for _, operand := range a.Operands {
		operandInput, err := operand.ToGqlInput()
		if err != nil {
			return nil, fmt.Errorf("operand to gql input: %w", err)
		}

		operands = append(operands, *operandInput)
	}

	return &accessGovernanceType.AbacComparisonExpressionAggregatorInput{
		Operator: accessGovernanceType.BinaryExpressionAggregatorOperator(a.Operator.String()),
		Operands: operands,
	}, nil
}

//go:generate go run github.com/raito-io/enumer -type=UnaryOperator -values -gqlgen -yaml -json -trimprefix=UnaryOperator
type UnaryOperator int

const (
	UnaryOperatorNot UnaryOperator = iota
)

type UnaryExpression struct {
	Operator UnaryOperator    `json:"operator"`
	Operand  BinaryExpression `json:"expression"`
}

func (u UnaryExpression) ToGqlInput() (*accessGovernanceType.AbacComparisonExpressionUnaryExpressionInput, error) {
	operandInput, err := u.Operand.ToGqlInput()
	if err != nil {
		return nil, fmt.Errorf("operand to gql input: %w", err)
	}

	return &accessGovernanceType.AbacComparisonExpressionUnaryExpressionInput{
		Operator: accessGovernanceType.BinaryExpressionUnaryExpressionOperator(u.Operator.String()),
		Operand:  *operandInput,
	}, nil
}
