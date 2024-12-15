package dto

import (
	"fmt"
	"github.com/go-streamline/interfaces/definitions"
	"github.com/google/uuid"
	"strings"
)

// Parsing Logic for Flow String

type TokenType int

const (
	TokenIdentifier TokenType = iota
	TokenArrow
	TokenComma
	TokenLBracket
	TokenRBracket
	TokenEOF
)

type Token struct {
	typ   TokenType
	value string
}

type Tokenizer struct {
	input string
	pos   int
}

func (t *Tokenizer) tokenize() ([]Token, error) {
	var tokens []Token
	input := t.input
	for t.pos < len(input) {
		switch {
		case strings.HasPrefix(input[t.pos:], "->"):
			tokens = append(tokens, Token{typ: TokenArrow, value: "->"})
			t.pos += 2
		case input[t.pos] == '[':
			tokens = append(tokens, Token{typ: TokenLBracket, value: "["})
			t.pos++
		case input[t.pos] == ']':
			tokens = append(tokens, Token{typ: TokenRBracket, value: "]"})
			t.pos++
		case input[t.pos] == ',':
			tokens = append(tokens, Token{typ: TokenComma, value: ","})
			t.pos++
		case isWhitespace(input[t.pos]):
			t.pos++
		default:
			start := t.pos
			for t.pos < len(input) && isIdentifierChar(input[t.pos]) {
				t.pos++
			}
			if start == t.pos {
				return nil, fmt.Errorf("invalid character at position %d", t.pos)
			}
			tokens = append(tokens, Token{typ: TokenIdentifier, value: input[start:t.pos]})
		}
	}
	tokens = append(tokens, Token{typ: TokenEOF})
	return tokens, nil
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

func isIdentifierChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

type Parser struct {
	tokens []Token
	pos    int
	nodes  map[string]*ProcessorNode
}

type ProcessorNode struct {
	Name string
	Next []*ProcessorNode
}

type FlowNode struct {
	StartNodes []*ProcessorNode
	EndNodes   []*ProcessorNode
}

func (p *Parser) parseFlow() (*FlowNode, error) {
	return p.parseSequence()
}

func (p *Parser) parseSequence() (*FlowNode, error) {
	node, err := p.parseNode()
	if err != nil {
		return nil, err
	}
	if p.match(TokenArrow) {
		nextSequence, err := p.parseSequence()
		if err != nil {
			return nil, err
		}
		// Link end nodes of node to start nodes of nextSequence
		for _, endNode := range node.EndNodes {
			endNode.Next = append(endNode.Next, nextSequence.StartNodes...)
		}
		return &FlowNode{
			StartNodes: node.StartNodes,
			EndNodes:   nextSequence.EndNodes,
		}, nil
	}
	return node, nil
}

func (p *Parser) parseNode() (*FlowNode, error) {
	if p.match(TokenIdentifier) {
		name := p.previous().value
		node := p.getOrCreateProcessorNode(name)
		return &FlowNode{
			StartNodes: []*ProcessorNode{node},
			EndNodes:   []*ProcessorNode{node},
		}, nil
	} else if p.match(TokenLBracket) {
		branches, err := p.parseBranches()
		if err != nil {
			return nil, err
		}
		if !p.match(TokenRBracket) {
			return nil, fmt.Errorf("expected ']', got %v", p.peek())
		}
		return branches, nil
	}
	return nil, fmt.Errorf("expected IDENTIFIER or '[', got %v", p.peek())
}

func (p *Parser) parseBranches() (*FlowNode, error) {
	branch, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	startNodes := branch.StartNodes
	endNodes := branch.EndNodes

	for p.match(TokenComma) {
		branch, err := p.parseSequence()
		if err != nil {
			return nil, err
		}
		startNodes = append(startNodes, branch.StartNodes...)
		endNodes = append(endNodes, branch.EndNodes...)
	}
	return &FlowNode{
		StartNodes: startNodes,
		EndNodes:   endNodes,
	}, nil
}

func (p *Parser) match(expected TokenType) bool {
	if p.check(expected) {
		p.pos++
		return true
	}
	return false
}

func (p *Parser) check(expected TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().typ == expected
}

func (p *Parser) isAtEnd() bool {
	return p.peek().typ == TokenEOF
}

func (p *Parser) peek() Token {
	return p.tokens[p.pos]
}

func (p *Parser) previous() Token {
	return p.tokens[p.pos-1]
}

func (p *Parser) getOrCreateProcessorNode(name string) *ProcessorNode {
	if node, exists := p.nodes[name]; exists {
		return node
	}
	node := &ProcessorNode{Name: name}
	p.nodes[name] = node
	return node
}

// FlowDTOToEntity converts a FlowDTO to a Flow entity.
func FlowDTOToEntity(flowDTO *FlowDTO) (*definitions.Flow, error) {
	flow := &definitions.Flow{
		ID:          flowDTO.ID,
		Name:        flowDTO.Name,
		Description: flowDTO.Description,
		Active:      flowDTO.Active,
	}

	// Convert ProcessorDTOs to entities and store in a map by name
	processorEntities := make(map[string]*definitions.SimpleProcessor)
	for _, processorDTO := range flowDTO.Processors {
		entity := ProcessorDTOToEntity(&processorDTO)
		processorEntities[processorDTO.Name] = entity
		flow.Processors = append(flow.Processors, entity) // Include all processors
	}

	// Convert TriggerProcessorDTOs to entities
	for _, triggerDTO := range flowDTO.TriggerProcessors {
		triggerEntity := TriggerProcessorDTOToEntity(&triggerDTO)
		flow.TriggerProcessors = append(flow.TriggerProcessors, triggerEntity)
	}

	// Parse the flow string
	tokenizer := &Tokenizer{input: flowDTO.Flow}
	tokens, err := tokenizer.tokenize()
	if err != nil {
		return nil, err
	}

	parser := &Parser{
		tokens: tokens,
		nodes:  make(map[string]*ProcessorNode),
	}
	_, err = parser.parseFlow()
	if err != nil {
		return nil, err
	}

	// Link NextProcessors by reference
	for name, node := range parser.nodes {
		entity := processorEntities[name]
		for _, nextNode := range node.Next {
			nextEntity, ok := processorEntities[nextNode.Name]
			if !ok {
				return nil, fmt.Errorf("unknown processor '%s' in flow", nextNode.Name)
			}
			entity.NextProcessors = append(entity.NextProcessors, nextEntity)
		}
	}

	// Identify FirstProcessors (not referenced by anyone)
	referenced := make(map[uuid.UUID]bool)
	for _, p := range flow.Processors {
		for _, np := range p.NextProcessors {
			referenced[np.ID] = true
		}
	}
	var firstProcessors []*definitions.SimpleProcessor
	for _, p := range flow.Processors {
		if !referenced[p.ID] {
			firstProcessors = append(firstProcessors, p)
		}
	}
	flow.FirstProcessors = firstProcessors

	return flow, nil
}
