/*
 * Copyright 2012-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package SpringCore

import (
	"go/token"
	"go/types"
	"strings"

	"github.com/go-spring/go-spring-parent/spring-const"
	"github.com/go-spring/go-spring-parent/spring-logger"
	"github.com/spf13/cast"
)

// Condition 定义一个判断条件
type Condition interface {
	// Matches 成功返回 true，失败返回 false
	Matches(ctx SpringContext) bool
}

// ConditionFunc 定义 Condition 接口 Matches 方法的类型
type ConditionFunc func(ctx SpringContext) bool

// functionCondition 基于 Matches 方法的 Condition 实现
type functionCondition struct {
	fn ConditionFunc
}

// NewFunctionCondition functionCondition 的构造函数
func NewFunctionCondition(fn ConditionFunc) *functionCondition {
	if fn == nil {
		SpringLogger.Panic("fn can't be nil")
	}
	return &functionCondition{fn}
}

// Matches 成功返回 true，失败返回 false
func (c *functionCondition) Matches(ctx SpringContext) bool {
	return c.fn(ctx)
}

// propertyCondition 基于属性值存在的 Condition 实现
type propertyCondition struct {
	name string
}

// NewPropertyCondition propertyCondition 的构造函数
func NewPropertyCondition(name string) *propertyCondition {
	return &propertyCondition{name}
}

// Matches 成功返回 true，失败返回 false
func (c *propertyCondition) Matches(ctx SpringContext) bool {
	return len(ctx.GetPrefixProperties(c.name)) > 0
}

// missingPropertyCondition 基于属性值不存在的 Condition 实现
type missingPropertyCondition struct {
	name string
}

// NewMissingPropertyCondition missingPropertyCondition 的构造函数
func NewMissingPropertyCondition(name string) *missingPropertyCondition {
	return &missingPropertyCondition{name}
}

// Matches 成功返回 true，失败返回 false
func (c *missingPropertyCondition) Matches(ctx SpringContext) bool {
	return len(ctx.GetPrefixProperties(c.name)) == 0
}

// propertyValueCondition 基于属性值匹配的 Condition 实现
type propertyValueCondition struct {
	name        string
	havingValue interface{}
}

// NewPropertyValueCondition propertyValueCondition 的构造函数
func NewPropertyValueCondition(name string, havingValue interface{}) *propertyValueCondition {
	return &propertyValueCondition{name, havingValue}
}

// Matches 成功返回 true，失败返回 false
func (c *propertyValueCondition) Matches(ctx SpringContext) bool {
	// 参考 /usr/local/go/src/go/types/eval_test.go 示例

	val, ok := ctx.GetDefaultProperty(c.name, "")
	if !ok { // 不存在直接返回 false
		return false
	}

	// 不是字符串则直接比较
	expectValue, ok := c.havingValue.(string)
	if !ok {
		return val == c.havingValue
	}

	// 字符串不是表达式的话则直接比较
	if ok = strings.Contains(expectValue, "$"); !ok {
		return val == expectValue
	}

	expr := strings.Replace(expectValue, "$", cast.ToString(val), -1)
	gotTv, err := types.Eval(token.NewFileSet(), nil, token.NoPos, expr)
	if err != nil {
		SpringLogger.Panic(err)
	}
	return gotTv.Value.String() == "true"
}

// beanCondition 基于 Bean 存在的 Condition 实现
type beanCondition struct {
	beanId string
}

// NewBeanCondition beanCondition 的构造函数
func NewBeanCondition(beanId string) *beanCondition {
	return &beanCondition{beanId}
}

// Matches 成功返回 true，失败返回 false
func (c *beanCondition) Matches(ctx SpringContext) bool {
	_, ok := ctx.FindBeanByName(c.beanId)
	return ok
}

// missingBeanCondition 基于 Bean 不能存在的 Condition 实现
type missingBeanCondition struct {
	beanId string
}

// NewMissingBeanCondition missingBeanCondition 的构造函数
func NewMissingBeanCondition(beanId string) *missingBeanCondition {
	return &missingBeanCondition{beanId}
}

// Matches 成功返回 true，失败返回 false
func (c *missingBeanCondition) Matches(ctx SpringContext) bool {
	_, ok := ctx.FindBeanByName(c.beanId)
	return !ok
}

// expressionCondition 基于表达式的 Condition 实现
type expressionCondition struct {
	expression string
}

// NewExpressionCondition expressionCondition 的构造函数
func NewExpressionCondition(expression string) *expressionCondition {
	return &expressionCondition{expression}
}

// Matches 成功返回 true，失败返回 false
func (c *expressionCondition) Matches(ctx SpringContext) bool {
	SpringLogger.Panic(SpringConst.UNIMPLEMENTED_METHOD)
	return false
}

// profileCondition 基于运行环境匹配的 Condition 实现
type profileCondition struct {
	profile string
}

// NewProfileCondition profileCondition 的构造函数
func NewProfileCondition(profile string) *profileCondition {
	return &profileCondition{
		profile: profile,
	}
}

// Matches 成功返回 true，失败返回 false
func (c *profileCondition) Matches(ctx SpringContext) bool {
	if c.profile != "" && c.profile != ctx.GetProfile() {
		return false
	}
	return true
}

// ConditionOp conditionNode 的计算方式
type ConditionOp int

const (
	ConditionDefault = ConditionOp(0) // 默认值
	ConditionOr      = ConditionOp(1) // 或
	ConditionAnd     = ConditionOp(2) // 且
	ConditionNone    = ConditionOp(3) // 非
)

// conditions 基于条件组的 Condition 实现
type conditions struct {
	op   ConditionOp
	cond []Condition
}

// NewConditions conditions 的构造函数
func NewConditions(op ConditionOp, cond ...Condition) *conditions {
	return &conditions{
		op:   op,
		cond: cond,
	}
}

// Matches 成功返回 true，失败返回 false
func (c *conditions) Matches(ctx SpringContext) bool {

	if len(c.cond) == 0 {
		SpringLogger.Panic("no condition")
	}

	switch c.op {
	case ConditionOr:
		for _, c0 := range c.cond {
			if c0.Matches(ctx) {
				return true
			}
		}
		return false
	case ConditionAnd:
		for _, c0 := range c.cond {
			if ok := c0.Matches(ctx); !ok {
				return false
			}
		}
		return true
	case ConditionNone:
		for _, c0 := range c.cond {
			if c0.Matches(ctx) {
				return false
			}
		}
		return true
	default:
		SpringLogger.Panic("error condition op mode")
	}

	return false
}

// conditionNode Condition 计算式的节点
type conditionNode struct {
	next *conditionNode // 下一个计算节点
	op   ConditionOp    // 计算方式
	cond Condition      // 条件
}

// newConditionNode conditionNode 的构造函数
func newConditionNode() *conditionNode {
	return &conditionNode{
		op: ConditionDefault,
	}
}

// Matches 成功返回 true，失败返回 false
func (c *conditionNode) Matches(ctx SpringContext) bool {

	if c.next != nil && c.next.cond == nil {
		SpringLogger.Panic("last op need a cond triggered")
	}

	if c.cond == nil && c.op == ConditionDefault {
		return true
	}

	if r := c.cond.Matches(ctx); c.next != nil {

		switch c.op {
		case ConditionOr: // or
			if r {
				return r
			} else {
				return c.next.Matches(ctx)
			}
		case ConditionAnd: // and
			if r {
				return c.next.Matches(ctx)
			} else {
				return false
			}
		default:
			SpringLogger.Panic("error condition op mode")
		}

	} else {
		return r
	}

	return false
}

// Conditional Condition 计算式
type Conditional struct {
	head *conditionNode
	curr *conditionNode
}

// NewConditional Conditional 的构造函数
func NewConditional() *Conditional {
	node := newConditionNode()
	return &Conditional{
		head: node,
		curr: node,
	}
}

// Empty 返回表达式是否为空
func (c *Conditional) Empty() bool {
	return c.head == c.curr
}

// Matches 成功返回 true，失败返回 false
func (c *Conditional) Matches(ctx SpringContext) bool {
	return c.head.Matches(ctx)
}

// Or c=a||b
func (c *Conditional) Or() *Conditional {
	node := newConditionNode()
	c.curr.op = ConditionOr
	c.curr.next = node
	c.curr = node
	return c
}

// And c=a&&b
func (c *Conditional) And() *Conditional {
	node := newConditionNode()
	c.curr.op = ConditionAnd
	c.curr.next = node
	c.curr = node
	return c
}

// OnCondition 设置一个 Condition
func (c *Conditional) OnCondition(cond Condition) *Conditional {
	if c.curr.cond != nil {
		c.And()
	}
	c.curr.cond = cond
	return c
}

// OnProperty 设置一个 propertyCondition
func (c *Conditional) OnProperty(name string) *Conditional {
	return c.OnCondition(NewPropertyCondition(name))
}

// OnMissingProperty 设置一个 missingPropertyCondition
func (c *Conditional) OnMissingProperty(name string) *Conditional {
	return c.OnCondition(NewMissingPropertyCondition(name))
}

// OnPropertyValue 设置一个 propertyValueCondition
func (c *Conditional) OnPropertyValue(name string, havingValue interface{}) *Conditional {
	return c.OnCondition(NewPropertyValueCondition(name, havingValue))
}

// OnBean 设置一个 beanCondition
func (c *Conditional) OnBean(beanId string) *Conditional {
	return c.OnCondition(NewBeanCondition(beanId))
}

// OnMissingBean 设置一个 missingBeanCondition
func (c *Conditional) OnMissingBean(beanId string) *Conditional {
	return c.OnCondition(NewMissingBeanCondition(beanId))
}

// OnExpression 设置一个 expressionCondition
func (c *Conditional) OnExpression(expression string) *Conditional {
	return c.OnCondition(NewExpressionCondition(expression))
}

// OnMatches 设置一个 functionCondition
func (c *Conditional) OnMatches(fn ConditionFunc) *Conditional {
	return c.OnCondition(NewFunctionCondition(fn))
}

// OnProfile 设置一个 profileCondition
func (c *Conditional) OnProfile(profile string) *Conditional {
	return c.OnCondition(NewProfileCondition(profile))
}
