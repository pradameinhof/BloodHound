// Copyright 2024 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package translate

import (
	"fmt"
	"log/slog"

	"github.com/specterops/bloodhound/cypher/models/cypher"

	"github.com/specterops/bloodhound/cypher/models/pgsql"
	"github.com/specterops/bloodhound/cypher/models/walk"
)

func (s *Translator) translateKindMatcher(kindMatcher *cypher.KindMatcher) error {
	if variable, isVariable := kindMatcher.Reference.(*cypher.Variable); !isVariable {
		return fmt.Errorf("expected variable for kind matcher reference but found type: %T", kindMatcher.Reference)
	} else if binding, resolved := s.scope.LookupString(variable.Symbol); !resolved {
		return fmt.Errorf("unable to find identifier %s", variable.Symbol)
	} else if kindIDs, err := s.kindMapper.MapKinds(s.ctx, kindMatcher.Kinds); err != nil {
		s.SetError(fmt.Errorf("failed to translate kinds: %w", err))
	} else {
		kindIDsLiteral := pgsql.NewLiteral(kindIDs, pgsql.Int2Array)

		switch binding.DataType {
		case pgsql.NodeComposite, pgsql.ExpansionRootNode, pgsql.ExpansionTerminalNode:
			s.treeTranslator.Push(pgsql.CompoundIdentifier{binding.Identifier, pgsql.ColumnKindIDs})
			s.treeTranslator.Push(kindIDsLiteral)

			if err := s.treeTranslator.PopPushOperator(s.scope, pgsql.OperatorPGArrayOverlap); err != nil {
				s.SetError(err)
			}

		case pgsql.EdgeComposite, pgsql.ExpansionEdge:
			s.treeTranslator.Push(pgsql.CompoundIdentifier{binding.Identifier, pgsql.ColumnKindID})
			s.treeTranslator.Push(pgsql.NewAnyExpressionHinted(kindIDsLiteral))

			if err := s.treeTranslator.PopPushOperator(s.scope, pgsql.OperatorEquals); err != nil {
				s.SetError(err)
			}

		default:
			return fmt.Errorf("unexpected kind matcher reference data type: %s", binding.DataType)
		}
	}

	return nil
}

func unwrapParenthetical(parenthetical pgsql.Expression) pgsql.Expression {
	next := parenthetical

	for next != nil {
		switch typedNext := next.(type) {
		case pgsql.Parenthetical:
			next = typedNext.Expression

		default:
			return next
		}
	}

	return parenthetical
}

func (s *Translator) translatePropertyLookup(lookup *cypher.PropertyLookup) error {
	if translatedAtom, err := s.treeTranslator.Pop(); err != nil {
		return err
	} else {
		switch typedTranslatedAtom := translatedAtom.(type) {
		case pgsql.Identifier:
			if fieldIdentifierLiteral, err := pgsql.AsLiteral(lookup.Symbols[0]); err != nil {
				return err
			} else {
				s.treeTranslator.Push(pgsql.CompoundIdentifier{typedTranslatedAtom, pgsql.ColumnProperties})
				s.treeTranslator.Push(fieldIdentifierLiteral)

				if err := s.treeTranslator.PopPushOperator(s.scope, pgsql.OperatorPropertyLookup); err != nil {
					return err
				}
			}

		case pgsql.FunctionCall:
			if fieldIdentifierLiteral, err := pgsql.AsLiteral(lookup.Symbols[0]); err != nil {
				return err
			} else if componentName, typeOK := fieldIdentifierLiteral.Value.(string); !typeOK {
				return fmt.Errorf("expected a string component name in translated literal but received type: %T", fieldIdentifierLiteral.Value)
			} else {
				switch typedTranslatedAtom.Function {
				case pgsql.FunctionCurrentDate, pgsql.FunctionLocalTime, pgsql.FunctionCurrentTime, pgsql.FunctionLocalTimestamp, pgsql.FunctionNow:
					switch componentName {
					case cypher.ITTCEpochSeconds:
						s.treeTranslator.Push(pgsql.FunctionCall{
							Function: pgsql.FunctionExtract,
							Parameters: []pgsql.Expression{pgsql.ProjectionFrom{
								Projection: []pgsql.SelectItem{
									pgsql.EpochIdentifier,
								},
								From: []pgsql.FromClause{{
									Source: translatedAtom,
								}},
							}},
							CastType: pgsql.Numeric,
						})

					case cypher.ITTCEpochMilliseconds:
						s.treeTranslator.Push(pgsql.NewBinaryExpression(
							pgsql.FunctionCall{
								Function: pgsql.FunctionExtract,
								Parameters: []pgsql.Expression{pgsql.ProjectionFrom{
									Projection: []pgsql.SelectItem{
										pgsql.EpochIdentifier,
									},
									From: []pgsql.FromClause{{
										Source: translatedAtom,
									}},
								}},
								CastType: pgsql.Numeric,
							},
							pgsql.OperatorMultiply,
							pgsql.NewLiteral(1000, pgsql.Int4),
						))

					default:
						return fmt.Errorf("unsupported date time instant type component %s from function call %s", componentName, typedTranslatedAtom.Function)
					}

				default:
					return fmt.Errorf("unsupported instant type component %s from function call %s", componentName, typedTranslatedAtom.Function)
				}
			}
		}
	}

	return nil
}

type PropertyLookup struct {
	Reference pgsql.CompoundIdentifier
	Field     string
}

func asPropertyLookup(expression pgsql.Expression) (*pgsql.BinaryExpression, bool) {
	switch typedExpression := expression.(type) {
	case pgsql.AnyExpression:
		// This is here to unwrap Any expressions that have been passed in as a property lookup. This is
		// common when dealing with array operators. In the future this check should be handled by the
		// caller to simplify the logic here.
		return asPropertyLookup(typedExpression.Expression)

	case *pgsql.BinaryExpression:
		return typedExpression, pgsql.OperatorIsPropertyLookup(typedExpression.Operator)
	}

	return nil, false
}

func decomposePropertyLookup(expression pgsql.Expression) (PropertyLookup, error) {
	if propertyLookup, isPropertyLookup := asPropertyLookup(expression); !isPropertyLookup {
		return PropertyLookup{}, fmt.Errorf("expected binary expression for property lookup decomposition but found type: %T", expression)
	} else if reference, typeOK := propertyLookup.LOperand.(pgsql.CompoundIdentifier); !typeOK {
		return PropertyLookup{}, fmt.Errorf("expected left operand for property lookup to be a compound identifier but found type: %T", propertyLookup.LOperand)
	} else if field, typeOK := propertyLookup.ROperand.(pgsql.Literal); !typeOK {
		return PropertyLookup{}, fmt.Errorf("expected right operand for property lookup to be a literal but found type: %T", propertyLookup.ROperand)
	} else if field.CastType != pgsql.Text {
		return PropertyLookup{}, fmt.Errorf("expected property lookup field a string literal but found data type: %s", field.CastType)
	} else if stringField, typeOK := field.Value.(string); !typeOK {
		return PropertyLookup{}, fmt.Errorf("expected property lookup field a string literal but found data type: %T", field)
	} else {
		return PropertyLookup{
			Reference: reference,
			Field:     stringField,
		}, nil
	}
}

func translateCypherAssignmentOperator(operator cypher.AssignmentOperator) (pgsql.Operator, error) {
	switch operator {
	case cypher.OperatorAssignment:
		return pgsql.OperatorAssignment, nil
	case cypher.OperatorLabelAssignment:
		return pgsql.OperatorKindAssignment, nil
	default:
		return pgsql.UnsetOperator, fmt.Errorf("unsupported assignment operator %s", operator)
	}
}

func ExtractSyntaxNodeReferences(root pgsql.SyntaxNode) (*pgsql.IdentifierSet, error) {
	dependencies := pgsql.NewIdentifierSet()

	return dependencies, walk.PgSQL(root, walk.NewSimpleVisitor[pgsql.SyntaxNode](
		func(node pgsql.SyntaxNode, errorHandler walk.CancelableErrorHandler) {
			switch typedNode := node.(type) {
			case pgsql.Identifier:
				// Filter for reserved identifiers
				if !pgsql.IsReservedIdentifier(typedNode) {
					dependencies.Add(typedNode)
				}

			case pgsql.CompoundIdentifier:
				identifier := typedNode.Root()

				if !pgsql.IsReservedIdentifier(identifier) {
					dependencies.Add(identifier)
				}
			}
		},
	))
}

func applyUnaryExpressionTypeHints(expression *pgsql.UnaryExpression) error {
	if propertyLookup, isPropertyLookup := asPropertyLookup(expression.Operand); isPropertyLookup {
		expression.Operand = rewritePropertyLookupOperator(propertyLookup, pgsql.Boolean)
	}

	return nil
}

func rewritePropertyLookupOperator(propertyLookup *pgsql.BinaryExpression, dataType pgsql.DataType) pgsql.Expression {
	if dataType.IsArrayType() {
		// Ensure that array conversions use JSONB
		propertyLookup.Operator = pgsql.OperatorJSONField

		return pgsql.FunctionCall{
			Function:   pgsql.FunctionJSONBToTextArray,
			Parameters: []pgsql.Expression{propertyLookup},
			CastType:   dataType,
		}
	}

	switch dataType {
	case pgsql.Text:
		propertyLookup.Operator = pgsql.OperatorJSONTextField
		return propertyLookup

	case pgsql.Date, pgsql.TimestampWithoutTimeZone, pgsql.TimestampWithTimeZone, pgsql.TimeWithoutTimeZone, pgsql.TimeWithTimeZone:
		propertyLookup.Operator = pgsql.OperatorJSONTextField
		return pgsql.NewTypeCast(propertyLookup, dataType)

	case pgsql.UnknownDataType:
		propertyLookup.Operator = pgsql.OperatorJSONTextField
		return propertyLookup

	default:
		propertyLookup.Operator = pgsql.OperatorJSONTextField
		return pgsql.NewTypeCast(propertyLookup, dataType)
	}
}

func GetTypeHint(expression pgsql.Expression) (pgsql.DataType, bool) {
	if typeHintedExpression, isTypeHinted := expression.(pgsql.TypeHinted); isTypeHinted {
		return typeHintedExpression.TypeHint(), true
	}

	return pgsql.UnsetDataType, false
}

func inferBinaryExpressionType(expression *pgsql.BinaryExpression) (pgsql.DataType, error) {
	var (
		leftHint, isLeftHinted   = GetTypeHint(expression.LOperand)
		rightHint, isRightHinted = GetTypeHint(expression.ROperand)
	)

	if isLeftHinted {
		if isRightHinted {
			if higherLevelHint, matchesOrConverts := leftHint.OperatorResultType(rightHint, expression.Operator); !matchesOrConverts {
				return pgsql.UnsetDataType, fmt.Errorf("left and right operands for binary expression \"%s\" are not compatible: %s != %s", expression.Operator, leftHint, rightHint)
			} else {
				return higherLevelHint, nil
			}
		} else if inferredRightHint, err := InferExpressionType(expression.ROperand); err != nil {
			return pgsql.UnsetDataType, err
		} else if inferredRightHint == pgsql.UnknownDataType {
			// Assume the right side is convertable and return the left operand hint
			return leftHint, nil
		} else if upcastHint, matchesOrConverts := leftHint.OperatorResultType(inferredRightHint, expression.Operator); !matchesOrConverts {
			return pgsql.UnsetDataType, fmt.Errorf("left and right operands for binary expression \"%s\" are not compatible: %s != %s", expression.Operator, leftHint, inferredRightHint)
		} else {
			return upcastHint, nil
		}
	} else if isRightHinted {
		// There's no left type, attempt to infer it
		if inferredLeftHint, err := InferExpressionType(expression.LOperand); err != nil {
			return pgsql.UnsetDataType, err
		} else if inferredLeftHint == pgsql.UnknownDataType {
			// Assume the right side is convertable and return the left operand hint
			return rightHint, nil
		} else if upcastHint, matchesOrConverts := rightHint.OperatorResultType(inferredLeftHint, expression.Operator); !matchesOrConverts {
			return pgsql.UnsetDataType, fmt.Errorf("left and right operands for binary expression \"%s\" are not compatible: %s != %s", expression.Operator, rightHint, inferredLeftHint)
		} else {
			return upcastHint, nil
		}
	} else {
		// If neither side has specific type information then check the operator to see if it implies some type
		// hinting before resorting to inference
		switch expression.Operator {
		case pgsql.OperatorCypherStartsWith, pgsql.OperatorCypherContains, pgsql.OperatorCypherEndsWith:
			// String operations imply the operands must be text
			return pgsql.Text, nil

		case pgsql.OperatorAnd, pgsql.OperatorOr:
			// Boolean operators that the operands must be boolean
			return pgsql.Boolean, nil

		default:
			// The operator does not imply specific type information onto the operands. Attempt to infer any
			// information as a last ditch effort to type the AST nodes
			if inferredLeftHint, err := InferExpressionType(expression.LOperand); err != nil {
				return pgsql.UnsetDataType, err
			} else if inferredRightHint, err := InferExpressionType(expression.ROperand); err != nil {
				return pgsql.UnsetDataType, err
			} else if inferredLeftHint == pgsql.UnknownDataType && inferredRightHint == pgsql.UnknownDataType {
				// Unable to infer any type information, this may be resolved elsewhere so this is not explicitly
				// an error condition
				return pgsql.UnknownDataType, nil
			} else if higherLevelHint, matchesOrConverts := inferredLeftHint.OperatorResultType(inferredRightHint, expression.Operator); !matchesOrConverts {
				return pgsql.UnsetDataType, fmt.Errorf("left and right operands for binary expression \"%s\" are not compatible: %s != %s", expression.Operator, inferredLeftHint, inferredRightHint)
			} else {
				return higherLevelHint, nil
			}
		}
	}
}

func InferExpressionType(expression pgsql.Expression) (pgsql.DataType, error) {
	switch typedExpression := expression.(type) {
	case pgsql.Identifier, pgsql.RowColumnReference:
		return pgsql.UnknownDataType, nil

	case pgsql.CompoundIdentifier:
		if len(typedExpression) != 2 {
			return pgsql.UnsetDataType, fmt.Errorf("expected a compound identifier to have only 2 components but found: %d", len(typedExpression))
		}

		// Infer type information for well known column names
		switch typedExpression[1] {
		// TODO: Graph ID should be int2
		case pgsql.ColumnGraphID, pgsql.ColumnID, pgsql.ColumnStartID, pgsql.ColumnEndID:
			return pgsql.Int8, nil

		case pgsql.ColumnKindID:
			return pgsql.Int2, nil

		case pgsql.ColumnKindIDs:
			return pgsql.Int2Array, nil

		case pgsql.ColumnProperties:
			return pgsql.JSONB, nil

		default:
			return pgsql.UnknownDataType, nil
		}

	case pgsql.TypeHinted:
		return typedExpression.TypeHint(), nil

	case *pgsql.BinaryExpression:
		switch typedExpression.Operator {
		case pgsql.OperatorJSONTextField:
			// Text field lookups could be text or an unknown lookup - reduce it to an unknown type
			return pgsql.UnknownDataType, nil

		case pgsql.OperatorPropertyLookup, pgsql.OperatorJSONField:
			// This is unknown, not unset meaning that it can be re-cast by future inference inspections
			return pgsql.UnknownDataType, nil

		case pgsql.OperatorAnd, pgsql.OperatorOr, pgsql.OperatorEquals, pgsql.OperatorGreaterThan, pgsql.OperatorGreaterThanOrEqualTo,
			pgsql.OperatorLessThan, pgsql.OperatorLessThanOrEqualTo, pgsql.OperatorIn, pgsql.OperatorJSONBFieldExists,
			pgsql.OperatorLike, pgsql.OperatorILike, pgsql.OperatorPGArrayOverlap:
			return pgsql.Boolean, nil

		default:
			return inferBinaryExpressionType(typedExpression)
		}

	case pgsql.Parenthetical:
		return InferExpressionType(typedExpression.Expression)

	default:
		slog.Info(fmt.Sprintf("unable to infer type hint for expression type: %T", expression))
		return pgsql.UnknownDataType, nil
	}
}

func lookupRequiresElementType(typeHint pgsql.DataType, operator pgsql.Operator, otherOperand pgsql.SyntaxNode) bool {
	if typeHint.IsArrayType() {
		switch operator {
		case pgsql.OperatorIn:
			return true
		}

		switch otherOperand.(type) {
		case pgsql.AnyExpression:
			return true
		}
	}

	return false
}

func TypeCastExpression(expression pgsql.Expression, dataType pgsql.DataType) (pgsql.Expression, error) {
	if propertyLookup, isPropertyLookup := asPropertyLookup(expression); isPropertyLookup {
		var lookupTypeHint = dataType

		if lookupRequiresElementType(dataType, propertyLookup.Operator, propertyLookup.ROperand) {
			// Take the base type of the array type hint: <unit> in <collection>
			lookupTypeHint = dataType.ArrayBaseType()
		}

		return rewritePropertyLookupOperator(propertyLookup, lookupTypeHint), nil
	}

	return pgsql.NewTypeCast(expression, dataType), nil
}

func rewritePropertyLookupOperands(expression *pgsql.BinaryExpression) error {
	var (
		leftPropertyLookup, hasLeftPropertyLookup   = asPropertyLookup(expression.LOperand)
		rightPropertyLookup, hasRightPropertyLookup = asPropertyLookup(expression.ROperand)
	)

	// Ensure that direct property comparisons prefer JSONB - JSONB
	if hasLeftPropertyLookup && hasRightPropertyLookup {
		leftPropertyLookup.Operator = pgsql.OperatorJSONField
		rightPropertyLookup.Operator = pgsql.OperatorJSONField

		return nil
	}

	if hasLeftPropertyLookup {
		// This check exists here to prevent from overwriting a property lookup that's part of a <value> in <list>
		// binary expression. This may want for better ergonomics in the future
		if anyExpression, isAnyExpression := expression.ROperand.(*pgsql.AnyExpression); isAnyExpression {
			expression.LOperand = rewritePropertyLookupOperator(leftPropertyLookup, anyExpression.CastType.ArrayBaseType())
		} else if rOperandTypeHint, err := InferExpressionType(expression.ROperand); err != nil {
			return err
		} else {
			switch expression.Operator {
			case pgsql.OperatorIn:
				expression.LOperand = rewritePropertyLookupOperator(leftPropertyLookup, rOperandTypeHint.ArrayBaseType())

			case pgsql.OperatorCypherStartsWith, pgsql.OperatorCypherEndsWith, pgsql.OperatorCypherContains, pgsql.OperatorRegexMatch:
				expression.LOperand = rewritePropertyLookupOperator(leftPropertyLookup, pgsql.Text)

			default:
				expression.LOperand = rewritePropertyLookupOperator(leftPropertyLookup, rOperandTypeHint)
			}
		}
	}

	if hasRightPropertyLookup {
		if lOperandTypeHint, err := InferExpressionType(expression.LOperand); err != nil {
			return err
		} else {
			switch expression.Operator {
			case pgsql.OperatorIn:
				if arrayType, err := lOperandTypeHint.ToArrayType(); err != nil {
					return err
				} else {
					expression.ROperand = rewritePropertyLookupOperator(rightPropertyLookup, arrayType)
				}

			case pgsql.OperatorCypherStartsWith, pgsql.OperatorCypherEndsWith, pgsql.OperatorCypherContains, pgsql.OperatorRegexMatch:
				expression.ROperand = rewritePropertyLookupOperator(rightPropertyLookup, pgsql.Text)

			default:
				expression.ROperand = rewritePropertyLookupOperator(rightPropertyLookup, lOperandTypeHint)
			}
		}
	}

	return nil
}

func newFunctionCallComparatorError(functionCall pgsql.FunctionCall, operator pgsql.Operator, comparisonType pgsql.DataType) error {
	switch functionCall.Function {
	case pgsql.FunctionCoalesce:
		// This is a specific error statement for coalesce statements. These statements have ill-defined
		// type conversion semantics in Cypher. As such, exposing the type specificity of coalesce to the
		// user as a distinct error will help reduce the surprise of running on a non-Neo4j substrate.
		return fmt.Errorf("coalesce has type %s but is being compared against type %s - ensure that all arguments in the coalesce function match the type of the other side of the comparison", functionCall.CastType, comparisonType)
	}

	return nil
}

func applyTypeFunctionLikeTypeHints(expression *pgsql.BinaryExpression) error {
	switch typedLOperand := expression.LOperand.(type) {
	case pgsql.AnyExpression:
		if rOperandTypeHint, err := InferExpressionType(expression.ROperand); err != nil {
			return err
		} else {
			// In an any-expression where the type of the any-expression is unknown, attempt to infer it
			if !typedLOperand.CastType.IsKnown() {
				if rOperandArrayTypeHint, err := rOperandTypeHint.ToArrayType(); err != nil {
					return err
				} else {
					typedLOperand.CastType = rOperandArrayTypeHint
					expression.LOperand = typedLOperand
				}
			} else if !rOperandTypeHint.IsKnown() {
				expression.ROperand = pgsql.NewTypeCast(expression.ROperand, typedLOperand.CastType.ArrayBaseType())
			} else {
				// Validate against the array base type of the any-expression
				lOperandBaseType := typedLOperand.CastType.ArrayBaseType()

				if !lOperandBaseType.IsComparable(rOperandTypeHint, expression.Operator) {
					return fmt.Errorf("function call has return signature of type %s but is being compared using operator %s against type %s", typedLOperand.CastType, expression.Operator, rOperandTypeHint)
				}
			}
		}

	case pgsql.FunctionCall:
		if rOperandTypeHint, err := InferExpressionType(expression.ROperand); err != nil {
			return err
		} else {
			if !typedLOperand.CastType.IsKnown() {
				typedLOperand.CastType = rOperandTypeHint
				expression.LOperand = typedLOperand
			}

			if pgsql.OperatorIsComparator(expression.Operator) && !typedLOperand.CastType.IsComparable(rOperandTypeHint, expression.Operator) {
				return newFunctionCallComparatorError(typedLOperand, expression.Operator, rOperandTypeHint)
			}
		}
	}

	switch typedROperand := expression.ROperand.(type) {
	case pgsql.AnyExpression:
		if lOperandTypeHint, err := InferExpressionType(expression.LOperand); err != nil {
			return err
		} else {
			// In an any-expression where the type of the any-expression is unknown, attempt to infer it
			if !typedROperand.CastType.IsKnown() {
				if !lOperandTypeHint.IsKnown() {
					// If the left operand has no type information then assume this is a castable any array
					typedROperand.CastType = pgsql.AnyArray
				} else if rOperandArrayTypeHint, err := lOperandTypeHint.ToArrayType(); err != nil {
					return err
				} else {
					typedROperand.CastType = rOperandArrayTypeHint
					expression.ROperand = typedROperand
				}
			} else if !lOperandTypeHint.IsKnown() {
				expression.LOperand = pgsql.NewTypeCast(expression.LOperand, typedROperand.CastType.ArrayBaseType())
			} else {
				// Validate against the array base type of the any-expression
				rOperandBaseType := typedROperand.CastType.ArrayBaseType()

				if !typedROperand.CastType.IsComparable(lOperandTypeHint, expression.Operator) && !rOperandBaseType.IsComparable(lOperandTypeHint, expression.Operator) {
					return fmt.Errorf("function call has return signature of type %s but is being compared using operator %s against type %s", typedROperand.CastType, expression.Operator, lOperandTypeHint)
				}
			}
		}

	case pgsql.FunctionCall:
		if lOperandTypeHint, err := InferExpressionType(expression.LOperand); err != nil {
			return err
		} else {
			if !typedROperand.CastType.IsKnown() {
				typedROperand.CastType = lOperandTypeHint
				expression.ROperand = typedROperand
			} else if !lOperandTypeHint.IsKnown() {
				expression.LOperand = pgsql.NewTypeCast(expression.LOperand, typedROperand.CastType.ArrayBaseType())
			} else if pgsql.OperatorIsComparator(expression.Operator) && !typedROperand.CastType.IsComparable(lOperandTypeHint, expression.Operator) {
				return newFunctionCallComparatorError(typedROperand, expression.Operator, lOperandTypeHint)
			}
		}
	}

	return nil
}

func applyBinaryExpressionTypeHints(expression *pgsql.BinaryExpression) error {
	switch expression.Operator {
	case pgsql.OperatorPropertyLookup:
		// Don't directly hint property lookups but replace the operator with the JSON operator
		expression.Operator = pgsql.OperatorJSONTextField
		return nil
	}

	if err := rewritePropertyLookupOperands(expression); err != nil {
		return err
	}

	return applyTypeFunctionLikeTypeHints(expression)
}

type Builder struct {
	stack []pgsql.Expression
}

func NewExpressionTreeBuilder() *Builder {
	return &Builder{}
}

func (s *Builder) Depth() int {
	return len(s.stack)
}

func (s *Builder) IsEmpty() bool {
	return len(s.stack) == 0
}

func (s *Builder) Pop() (pgsql.Expression, error) {
	next := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]

	switch typedNext := next.(type) {
	case *pgsql.UnaryExpression:
		if err := applyUnaryExpressionTypeHints(typedNext); err != nil {
			return nil, err
		}

	case *pgsql.BinaryExpression:
		if err := applyBinaryExpressionTypeHints(typedNext); err != nil {
			return nil, err
		}
	}

	return next, nil
}

func (s *Builder) Peek() pgsql.Expression {
	return s.stack[len(s.stack)-1]
}

func (s *Builder) Push(expression pgsql.Expression) {
	s.stack = append(s.stack, expression)
}

type ExpressionTreeBuilder interface {
	Pop() (pgsql.Expression, error)
	Peek() pgsql.Expression
	Push(expression pgsql.Expression)
}

func PopFromBuilderAs[T any](builder ExpressionTreeBuilder) (T, error) {
	var empty T

	if value, err := builder.Pop(); err != nil {
		return empty, err
	} else if typed, isType := value.(T); isType {
		return typed, nil
	} else {
		return empty, fmt.Errorf("unable to convert type %T as %T", value, empty)
	}
}

func ConjoinExpressions(expressions []pgsql.Expression) (pgsql.Expression, error) {
	var conjoined pgsql.Expression

	for _, expression := range expressions {
		if expression == nil {
			continue
		}

		if conjoined == nil {
			conjoined = expression
			continue
		}

		conjoinedBinaryExpression := pgsql.NewBinaryExpression(conjoined, pgsql.OperatorAnd, expression)

		if err := applyBinaryExpressionTypeHints(conjoinedBinaryExpression); err != nil {
			return nil, err
		}

		conjoined = conjoinedBinaryExpression
	}

	return conjoined, nil
}

type ExpressionTreeTranslator struct {
	IdentifierConstraints *ConstraintTracker

	projectionConstraints []*Constraint
	treeBuilder           *Builder
	parentheticalDepth    int
	disjunctionDepth      int
	conjunctionDepth      int
}

func NewExpressionTreeTranslator() *ExpressionTreeTranslator {
	return &ExpressionTreeTranslator{
		IdentifierConstraints: NewConstraintTracker(),
		treeBuilder:           NewExpressionTreeBuilder(),
	}
}

func (s *ExpressionTreeTranslator) Consume(identifier pgsql.Identifier) (*Constraint, error) {
	return s.IdentifierConstraints.ConsumeSet(pgsql.AsIdentifierSet(identifier))
}

func (s *ExpressionTreeTranslator) ConsumeSet(identifierSet *pgsql.IdentifierSet) (*Constraint, error) {
	return s.IdentifierConstraints.ConsumeSet(identifierSet)
}

func (s *ExpressionTreeTranslator) ConsumeAll() (*Constraint, error) {
	if constraint, err := s.IdentifierConstraints.ConsumeAll(); err != nil {
		return nil, err
	} else {
		constraintExpressions := []pgsql.Expression{constraint.Expression}

		for _, projectionConstraint := range s.projectionConstraints {
			constraint.Dependencies.MergeSet(projectionConstraint.Dependencies)
			constraintExpressions = append(constraintExpressions, projectionConstraint.Expression)
		}

		if conjoined, err := ConjoinExpressions(constraintExpressions); err != nil {
			return nil, err
		} else {
			constraint.Expression = conjoined
		}

		return constraint, nil
	}
}

func (s *ExpressionTreeTranslator) Constrain(identifierSet *pgsql.IdentifierSet, expression pgsql.Expression) error {
	return s.IdentifierConstraints.Constrain(identifierSet, expression)
}

func (s *ExpressionTreeTranslator) ConstrainIdentifier(identifier pgsql.Identifier, expression pgsql.Expression) error {
	return s.Constrain(pgsql.AsIdentifierSet(identifier), expression)
}

func (s *ExpressionTreeTranslator) Depth() int {
	return s.treeBuilder.Depth()
}

func (s *ExpressionTreeTranslator) Push(expression pgsql.Expression) {
	s.treeBuilder.Push(expression)
}

func (s *ExpressionTreeTranslator) Peek() pgsql.Expression {
	return s.treeBuilder.Peek()
}

func (s *ExpressionTreeTranslator) Pop() (pgsql.Expression, error) {
	return s.treeBuilder.Pop()
}

func (s *ExpressionTreeTranslator) popExpressionAsConstraint() error {
	if nextExpression, err := s.Pop(); err != nil {
		return err
	} else if identifierDeps, err := ExtractSyntaxNodeReferences(nextExpression); err != nil {
		return err
	} else {
		if propertyLookup, isPropertyLookup := asPropertyLookup(nextExpression); isPropertyLookup {
			// If this is a bare property lookup rewrite it with the intended type of boolean
			nextExpression = rewritePropertyLookupOperator(propertyLookup, pgsql.Boolean)
		}

		return s.Constrain(identifierDeps, nextExpression)
	}
}

func (s *ExpressionTreeTranslator) PopRemainingExpressionsAsConstraints() error {
	// Pull the right operand only if one exists
	for !s.treeBuilder.IsEmpty() {
		if err := s.popExpressionAsConstraint(); err != nil {
			return err
		}
	}

	return nil
}

func (s *ExpressionTreeTranslator) ConstrainDisjointOperandPair() error {
	// Always expect a left operand
	if s.treeBuilder.IsEmpty() {
		return fmt.Errorf("expected at least one operand for constraint extraction")
	}

	if rightOperand, err := s.treeBuilder.Pop(); err != nil {
		return err
	} else if rightDependencies, err := ExtractSyntaxNodeReferences(rightOperand); err != nil {
		return err
	} else if s.treeBuilder.IsEmpty() {
		// If the tree builder is empty then this operand is at the top of the disjunction chain
		return s.Constrain(rightDependencies, rightOperand)
	} else if leftOperand, err := s.treeBuilder.Pop(); err != nil {
		return err
	} else {
		newOrExpression := pgsql.NewBinaryExpression(
			leftOperand,
			pgsql.OperatorOr,
			rightOperand,
		)

		if err := applyBinaryExpressionTypeHints(newOrExpression); err != nil {
			return err
		}

		// This operation may not be complete; push it back on the stack
		s.Push(newOrExpression)
		return nil
	}
}

func (s *ExpressionTreeTranslator) ConstrainConjoinedOperandPair() error {
	// Always expect a left operand
	if s.treeBuilder.IsEmpty() {
		return fmt.Errorf("expected at least one operand for constraint extraction")
	}

	if err := s.popExpressionAsConstraint(); err != nil {
		return err
	}

	return nil
}

func (s *ExpressionTreeTranslator) PopBinaryExpression(operator pgsql.Operator) (*pgsql.BinaryExpression, error) {
	if rightOperand, err := s.Pop(); err != nil {
		return nil, err
	} else if leftOperand, err := s.Pop(); err != nil {
		return nil, err
	} else {
		newBinaryExpression := pgsql.NewBinaryExpression(leftOperand, operator, rightOperand)
		return newBinaryExpression, applyBinaryExpressionTypeHints(newBinaryExpression)
	}
}

func rewriteIdentityOperands(scope *Scope, newExpression *pgsql.BinaryExpression) error {
	switch typedLOperand := newExpression.LOperand.(type) {
	case pgsql.Identifier:
		// If the left side is an identifier we need to inspect the type of the identifier bound in our scope
		if boundLOperand, bound := scope.Lookup(typedLOperand); !bound {
			return fmt.Errorf("unknown identifier %s", typedLOperand)
		} else {
			switch typedROperand := newExpression.ROperand.(type) {
			case pgsql.Identifier:
				// If the right side is an identifier, inspect to see if the identifiers are an entity comparison.
				// For example: match (n1)-[]->(n2) where n1 <> n2 return n2
				if boundROperand, bound := scope.Lookup(typedROperand); !bound {
					return fmt.Errorf("unknown identifier %s", typedROperand)
				} else {
					switch boundLOperand.DataType {
					case pgsql.NodeCompositeArray:
						return fmt.Errorf("unsupported pgsql.NodeCompositeArray")

					case pgsql.NodeComposite, pgsql.ExpansionRootNode, pgsql.ExpansionTerminalNode:
						switch boundROperand.DataType {
						case pgsql.NodeComposite, pgsql.ExpansionRootNode, pgsql.ExpansionTerminalNode:
							// If this is a node entity comparison of some kind then the AST must be rewritten to use identity properties
							newExpression.LOperand = pgsql.CompoundIdentifier{typedLOperand, pgsql.ColumnID}
							newExpression.ROperand = pgsql.CompoundIdentifier{typedROperand, pgsql.ColumnID}

						case pgsql.NodeCompositeArray:
							newExpression.LOperand = pgsql.CompoundIdentifier{typedLOperand, pgsql.ColumnID}
							newExpression.ROperand = pgsql.CompoundIdentifier{typedROperand, pgsql.ColumnID}

						default:
							return fmt.Errorf("invalid comparison between types %s and %s", boundLOperand.DataType, boundROperand.DataType)
						}

					case pgsql.EdgeCompositeArray:
						return fmt.Errorf("unsupported pgsql.EdgeCompositeArray")

					case pgsql.EdgeComposite, pgsql.ExpansionEdge:
						switch boundROperand.DataType {
						case pgsql.EdgeComposite, pgsql.ExpansionEdge:
							// If this is an edge entity comparison of some kind then the AST must be rewritten to use identity properties
							newExpression.LOperand = pgsql.CompoundIdentifier{typedLOperand, pgsql.ColumnID}
							newExpression.ROperand = pgsql.CompoundIdentifier{typedROperand, pgsql.ColumnID}

						case pgsql.EdgeCompositeArray:
							newExpression.LOperand = pgsql.CompoundIdentifier{typedLOperand, pgsql.ColumnID}
							newExpression.ROperand = pgsql.CompoundIdentifier{typedROperand, pgsql.ColumnID}

						default:
							return fmt.Errorf("invalid comparison between types %s and %s", boundLOperand.DataType, boundROperand.DataType)
						}

					case pgsql.PathComposite:
						return fmt.Errorf("comparison for path identifiers is unsupported")
					}
				}
			}
		}
	}

	return nil
}

func (s *ExpressionTreeTranslator) rewriteBinaryExpression(newExpression *pgsql.BinaryExpression) error {
	switch newExpression.Operator {
	case pgsql.OperatorCypherAdd:
		isConcatenationOperation := func(lOperandType, rOperandType pgsql.DataType) bool {
			// Any use of an array type automatically assumes concatenation
			if lOperandType.IsArrayType() || rOperandType.IsArrayType() {
				return true
			}

			switch lOperandType {
			case pgsql.Text:
				switch rOperandType {
				case pgsql.Text:
					return true
				}
			}

			return false
		}

		// In the case of the use of the cypher `+` operator we must attempt to disambiguate if the intent
		// is to concatenate or to perform an addition
		if lOperandType, err := InferExpressionType(newExpression.LOperand); err != nil {
			return err
		} else if rOperandType, err := InferExpressionType(newExpression.ROperand); err != nil {
			return err
		} else if isConcatenationOperation(lOperandType, rOperandType) {
			newExpression.Operator = pgsql.OperatorConcatenate
		}

		s.Push(newExpression)

	case pgsql.OperatorCypherContains:
		newExpression.Operator = pgsql.OperatorLike

		switch typedLOperand := newExpression.LOperand.(type) {
		case *pgsql.BinaryExpression:
			switch typedLOperand.Operator {
			case pgsql.OperatorPropertyLookup, pgsql.OperatorJSONField, pgsql.OperatorJSONTextField:
			default:
				return fmt.Errorf("unexpected operator %s for binary expression \"%s\" left operand", typedLOperand.Operator, newExpression.Operator)
			}
		}

		switch typedROperand := newExpression.ROperand.(type) {
		case pgsql.Literal:
			if rOperandDataType := typedROperand.TypeHint(); rOperandDataType != pgsql.Text {
				return fmt.Errorf("expected %s data type but found %s as right operand for operator %s", pgsql.Text, rOperandDataType, newExpression.Operator)
			} else if stringValue, isString := typedROperand.Value.(string); !isString {
				return fmt.Errorf("expected string but found %T as right operand for operator %s", typedROperand.Value, newExpression.Operator)
			} else {
				newExpression.ROperand = pgsql.NewLiteral("%"+stringValue+"%", rOperandDataType)
			}

		case pgsql.Parenthetical:
			if typeCastedROperand, err := TypeCastExpression(typedROperand, pgsql.Text); err != nil {
				return err
			} else {
				newExpression.ROperand = pgsql.NewBinaryExpression(
					pgsql.NewLiteral("%", pgsql.Text),
					pgsql.OperatorConcatenate,
					pgsql.NewBinaryExpression(
						typeCastedROperand,
						pgsql.OperatorConcatenate,
						pgsql.NewLiteral("%", pgsql.Text),
					),
				)
			}

		case *pgsql.BinaryExpression:
			if stringLiteral, err := pgsql.AsLiteral("%"); err != nil {
				return err
			} else {
				if pgsql.OperatorIsPropertyLookup(typedROperand.Operator) {
					typedROperand.Operator = pgsql.OperatorJSONTextField
				}

				newExpression.ROperand = pgsql.NewTypeCast(pgsql.NewBinaryExpression(
					stringLiteral,
					pgsql.OperatorConcatenate,
					pgsql.NewBinaryExpression(
						&pgsql.Parenthetical{
							Expression: typedROperand,
						},
						pgsql.OperatorConcatenate,
						stringLiteral,
					),
				), pgsql.Text)
			}

		default:
			newExpression.ROperand = pgsql.NewBinaryExpression(
				pgsql.NewLiteral("%", pgsql.Text),
				pgsql.OperatorConcatenate,
				pgsql.NewBinaryExpression(
					typedROperand,
					pgsql.OperatorConcatenate,
					pgsql.NewLiteral("%", pgsql.Text),
				),
			)
		}

		s.Push(newExpression)

	case pgsql.OperatorCypherRegexMatch:
		newExpression.Operator = pgsql.OperatorRegexMatch
		s.Push(newExpression)

	case pgsql.OperatorCypherStartsWith:
		newExpression.Operator = pgsql.OperatorLike

		switch typedLOperand := newExpression.LOperand.(type) {
		case *pgsql.BinaryExpression:
			switch typedLOperand.Operator {
			case pgsql.OperatorPropertyLookup, pgsql.OperatorJSONField, pgsql.OperatorJSONTextField:
			default:
				return fmt.Errorf("unexpected operator %s for binary expression \"%s\" left operand", typedLOperand.Operator, newExpression.Operator)
			}
		}

		switch typedROperand := newExpression.ROperand.(type) {
		case pgsql.Literal:
			if rOperandDataType := typedROperand.TypeHint(); rOperandDataType != pgsql.Text {
				return fmt.Errorf("expected %s data type but found %s as right operand for operator %s", pgsql.Text, rOperandDataType, newExpression.Operator)
			} else if stringValue, isString := typedROperand.Value.(string); !isString {
				return fmt.Errorf("expected string but found %T as right operand for operator %s", typedROperand.Value, newExpression.Operator)
			} else {
				newExpression.ROperand = pgsql.NewLiteral(stringValue+"%", rOperandDataType)
			}

		case pgsql.Parenthetical:
			if typeCastedROperand, err := TypeCastExpression(typedROperand, pgsql.Text); err != nil {
				return err
			} else {
				newExpression.ROperand = pgsql.NewBinaryExpression(
					typeCastedROperand,
					pgsql.OperatorConcatenate,
					pgsql.NewLiteral("%", pgsql.Text),
				)
			}

		case *pgsql.BinaryExpression:
			if stringLiteral, err := pgsql.AsLiteral("%"); err != nil {
				return err
			} else {
				if pgsql.OperatorIsPropertyLookup(typedROperand.Operator) {
					typedROperand.Operator = pgsql.OperatorJSONTextField
				}

				newExpression.ROperand = pgsql.NewTypeCast(pgsql.NewBinaryExpression(
					&pgsql.Parenthetical{
						Expression: typedROperand,
					},
					pgsql.OperatorConcatenate,
					stringLiteral,
				), pgsql.Text)
			}

		default:
			newExpression.ROperand = pgsql.NewBinaryExpression(
				typedROperand,
				pgsql.OperatorConcatenate,
				pgsql.NewLiteral("%", pgsql.Text),
			)
		}

		s.Push(newExpression)

	case pgsql.OperatorCypherEndsWith:
		newExpression.Operator = pgsql.OperatorLike

		switch typedLOperand := newExpression.LOperand.(type) {
		case *pgsql.BinaryExpression:
			switch typedLOperand.Operator {
			case pgsql.OperatorPropertyLookup, pgsql.OperatorJSONField, pgsql.OperatorJSONTextField:
			default:
				return fmt.Errorf("unexpected operator %s for binary expression \"%s\" left operand", typedLOperand.Operator, newExpression.Operator)
			}
		}

		switch typedROperand := newExpression.ROperand.(type) {
		case pgsql.Literal:
			if rOperandDataType := typedROperand.TypeHint(); rOperandDataType != pgsql.Text {
				return fmt.Errorf("expected %s data type but found %s as right operand for operator %s", pgsql.Text, rOperandDataType, newExpression.Operator)
			} else if stringValue, isString := typedROperand.Value.(string); !isString {
				return fmt.Errorf("expected string but found %T as right operand for operator %s", typedROperand.Value, newExpression.Operator)
			} else {
				newExpression.ROperand = pgsql.NewLiteral("%"+stringValue, rOperandDataType)
			}

		case pgsql.Parenthetical:
			if typeCastedROperand, err := TypeCastExpression(typedROperand, pgsql.Text); err != nil {
				return err
			} else {
				newExpression.ROperand = pgsql.NewBinaryExpression(
					pgsql.NewLiteral("%", pgsql.Text),
					pgsql.OperatorConcatenate,
					typeCastedROperand,
				)
			}

		case *pgsql.BinaryExpression:
			if pgsql.OperatorIsPropertyLookup(typedROperand.Operator) {
				typedROperand.Operator = pgsql.OperatorJSONTextField
			}

			newExpression.ROperand = pgsql.NewTypeCast(pgsql.NewBinaryExpression(
				pgsql.NewLiteral("%", pgsql.Text),
				pgsql.OperatorConcatenate,
				&pgsql.Parenthetical{
					Expression: typedROperand,
				},
			), pgsql.Text)

		default:
			newExpression.ROperand = pgsql.NewBinaryExpression(
				pgsql.NewLiteral("%", pgsql.Text),
				pgsql.OperatorConcatenate,
				typedROperand,
			)
		}

		s.Push(newExpression)

	case pgsql.OperatorIs:
		switch typedLOperand := newExpression.LOperand.(type) {
		case *pgsql.BinaryExpression:
			switch typedLOperand.Operator {
			case pgsql.OperatorPropertyLookup, pgsql.OperatorJSONField, pgsql.OperatorJSONTextField:
				// This is a null-check against a property. This should be rewritten using the JSON field exists
				// operator instead. It can be
				switch typedROperand := newExpression.ROperand.(type) {
				case pgsql.Literal:
					if typedROperand.Null {
						newExpression.Operator = pgsql.OperatorJSONBFieldExists
						newExpression.LOperand = typedLOperand.LOperand
						newExpression.ROperand = typedLOperand.ROperand
					}

					s.Push(pgsql.NewUnaryExpression(pgsql.OperatorNot, newExpression))
				}
			}
		}

	case pgsql.OperatorIsNot:
		switch typedLOperand := newExpression.LOperand.(type) {
		case *pgsql.BinaryExpression:
			switch typedLOperand.Operator {
			case pgsql.OperatorPropertyLookup, pgsql.OperatorJSONField, pgsql.OperatorJSONTextField:
				// This is a null-check against a property. This should be rewritten using the JSON field exists
				// operator instead. It can be
				switch typedROperand := newExpression.ROperand.(type) {
				case pgsql.Literal:
					if typedROperand.Null {
						newExpression.Operator = pgsql.OperatorJSONBFieldExists
						newExpression.LOperand = typedLOperand.LOperand
						newExpression.ROperand = typedLOperand.ROperand
					}

					s.Push(newExpression)
				}
			}
		}

	case pgsql.OperatorIn:
		newExpression.Operator = pgsql.OperatorEquals

		switch typedROperand := newExpression.ROperand.(type) {
		case pgsql.TypeCast:
			switch typedInnerOperand := typedROperand.Expression.(type) {
			case *pgsql.BinaryExpression:
				if propertyLookup, isPropertyLookup := asPropertyLookup(typedInnerOperand); isPropertyLookup {
					// Attempt to figure out the cast by looking at the left operand
					if leftHint, err := InferExpressionType(newExpression.LOperand); err != nil {
						return err
					} else if leftArrayHint, err := leftHint.ToArrayType(); err != nil {
						return err
					} else {
						// Ensure the lookup uses the JSONB type
						propertyLookup.Operator = pgsql.OperatorJSONField

						newExpression.ROperand = pgsql.NewAnyExpressionHinted(
							pgsql.FunctionCall{
								Function:   pgsql.FunctionJSONBToTextArray,
								Parameters: []pgsql.Expression{propertyLookup},
								CastType:   leftArrayHint,
							},
						)
					}
				}
			}

		case pgsql.TypeHinted:
			if lOperandTypeHint, err := InferExpressionType(newExpression.LOperand); err != nil {
				return err
			} else if lOperandTypeHint.IsArrayType() {
				newExpression.Operator = pgsql.OperatorPGArrayOverlap
			} else {
				newExpression.Operator = pgsql.OperatorEquals
				newExpression.ROperand = pgsql.NewAnyExpression(newExpression.ROperand, typedROperand.TypeHint())
			}

		default:
			// Attempt to figure out the cast by looking at the left operand
			if leftHint, err := InferExpressionType(newExpression.LOperand); err != nil {
				return err
			} else {
				newExpression.ROperand = pgsql.NewAnyExpression(newExpression.ROperand, leftHint)
			}
		}

		s.Push(newExpression)

	default:
		s.Push(newExpression)
	}

	return nil
}

func (s *ExpressionTreeTranslator) PopPushBinaryExpression(scope *Scope, operator pgsql.Operator) error {
	if newExpression, err := s.PopBinaryExpression(operator); err != nil {
		return err
	} else if err := rewriteIdentityOperands(scope, newExpression); err != nil {
		return err
	} else {
		return s.rewriteBinaryExpression(newExpression)
	}
}

func (s *ExpressionTreeTranslator) PushParenthetical() {
	s.Push(&pgsql.Parenthetical{})
	s.parentheticalDepth += 1
}

func (s *ExpressionTreeTranslator) PopParenthetical() (*pgsql.Parenthetical, error) {
	s.parentheticalDepth -= 1
	return PopFromBuilderAs[*pgsql.Parenthetical](s)
}

func (s *ExpressionTreeTranslator) PushOperator(operator pgsql.Operator) {
	// Track this operator for expression tree extraction
	switch operator {
	case pgsql.OperatorAnd:
		s.conjunctionDepth += 1

	case pgsql.OperatorOr:
		s.disjunctionDepth += 1
	}
}

func (s *ExpressionTreeTranslator) PopPushOperator(scope *Scope, operator pgsql.Operator) error {
	// Track this operator for expression tree extraction and look to see if it's a candidate for rewriting
	switch operator {
	case pgsql.OperatorAnd:
		if s.parentheticalDepth == 0 && s.disjunctionDepth == 0 {
			return s.ConstrainConjoinedOperandPair()
		}

		s.conjunctionDepth -= 1

	case pgsql.OperatorOr:
		if s.parentheticalDepth == 0 && s.conjunctionDepth == 0 {
			return s.ConstrainDisjointOperandPair()
		}

		s.disjunctionDepth -= 1
	}

	return s.PopPushBinaryExpression(scope, operator)
}
