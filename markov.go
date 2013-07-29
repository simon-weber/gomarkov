package markov

import (
    "fmt"
    "errors"
    "strings"
    "math/rand"
)

const NGrams = 2

// The number of individual suffixes considered during getRandom.
const NumSamples = 3

// Token represents one of three things: a start token, 
// and end token, or a string literal.
type Token struct {
    isStart bool
    isEnd bool
    value string // undefined if isStart || isEnd
}
func NewToken(s string) *Token{
    return &Token{value: s}
}

func (t Token) String() string {
    outputVal := fmt.Sprintf("\"%s\"", t.value)

    if t.isStart{
        outputVal = "<start>"
    } else if t.isEnd{
        outputVal = "<end>"
    }

	return fmt.Sprintf("Tok{%s}", outputVal)
}

// Chain implements a reader-optimized markov text generator.
type Chain struct {
    chains map[[NGrams]Token] *Suffixes

    numBuffered int

    tokenize func(string) []Token // must return at least [start, end]

    // by default, we don't normalize.
    // not defaulting to identity allows avoiding call overhead
    useNormalize bool
    normalize func(string) string
}

func (c Chain) String() string{
    return fmt.Sprintf("%s", c.chains)
}

func NewChain() *Chain {
    return &Chain{chains: map[[NGrams]Token] *Suffixes{}, tokenize: WhitespaceTokenize,
    useNormalize: false}
}

func NewCustomChain(tokenize func(string) []Token, normalize func(string) string) *Chain {
    return &Chain{chains: map[[NGrams]Token] *Suffixes{}, tokenize: tokenize,
    useNormalize: true, normalize: normalize}
}

/*
Respond creates a response from the current Chain state
which will contain between minLen and maxLen tokens (not including start/end tokens).

The output will attempt to seed from a random ngram in the input when possible.
Otherwise, it will seed from a random starting ngram in the Chain state.

To create output not seeded from input, pass "" as input.

If Update has not been called for this Chain, an error will be returned.
*/
func (c *Chain) Respond(input string, minLen int, maxLen int) (response string, err error){
    // init with not-really-random ngram
    response = ""
    err = nil

    if len(c.chains) == 0 {
        err = errors.New("cannot Respond with no chains built")
        return
    }
    if c.useNormalize {
        input = c.normalize(input)
    }


    tokens := c.tokenize(input)
    var responseToks []Token = nil

    if len(tokens) > NGrams{
        // try to seed with an ngram from the input

        ngram := [NGrams]Token{}

        // try ngrams in a random order
        indexOrder := rand.Perm(len(tokens) - NGrams + 1)
        for _, i := range indexOrder{
            fillNGram(&ngram, tokens, i)

            _, present := c.chains[ngram]
            if present{
                toks := c.respond(ngram, maxLen)
                if len(toks) > minLen{
                    responseToks = toks
                    break
                }
            }
        }
    }

    if responseToks == nil{
        // seed from a randomish ngram.
        for ngram := range c.chains{
            toks := c.respond(ngram, maxLen)
            if len(toks) > minLen{
                responseToks = toks
                break
            }
        }
    }

    if responseToks == nil{
        err = errors.New("could not generate response within constraints")
        return
    }

    response = joinTokens(responseToks)
    return
}

func joinTokens(toks []Token) string{
    strs := make([]string, 0, len(toks))
    for _, tok := range toks{
        if !(tok.isStart || tok.isEnd){
            strs = append(strs, tok.value)
        }
    }

    return strings.Join(strs, " ")
}

// start/end will not be included.
func (c *Chain) respond(seed [NGrams]Token, maxLen int) []Token{
    responseToks := make([]Token, 0, maxLen)
    ngram := seed

    for i:= 0; i < maxLen; i++{
        responseToks = append(responseToks, ngram[0])
        if ngram[0].isStart || ngram[0].isEnd{
            // don't count empty tokens towards length
            i--
        }

        suffixes, present := c.chains[ngram]
        if !present{
            break
        }
        shift(&ngram, suffixes.getRandom())
    }

    withoutSpecials := make([]Token, 0, len(responseToks))
    for _, tok := range responseToks{
        if !(tok.isStart || tok.isEnd){
            withoutSpecials = append(withoutSpecials, tok)
        }
    }

    return withoutSpecials

}

/*
Update tokenizes and normalizes the input line, then adds it to the Chain state.
*/
func (c *Chain) Update(line string){
    // data that doesn't fit evenly into an ngram is discarded.
    if c.useNormalize{
        line = c.normalize(line)
    }

    tokens := c.tokenize(line)

    if len(tokens) <= NGrams{
        return // not enough to fill an ngram
    }

    ngram := [NGrams]Token{}
    fillNGram(&ngram, tokens, 0)

    for i := 0; i < len(tokens) - NGrams; i++{
        suffix := tokens[i + NGrams]

        suffixes, present := c.chains[ngram]
        if !present{
            suffixes = NewSuffixes()
            c.chains[ngram] = suffixes
        }

        suffixes.Update(suffix)
        shift(&ngram, suffix)
    }
}

// shift the sequence left by one.
func shift(tokens *[NGrams]Token, suffix Token){
    for i := 0; i < NGrams - 1; i++{
        tokens[i] = tokens[i+1]
    }
    tokens[len(tokens) - 1] = suffix
}

func fillNGram(ngram *[NGrams]Token, ar []Token, from int){
    for i := 0; i < NGrams; i++{
        ngram[i] = ar[from + i]
    }
}

/*
WhitespaceTokenize splits tokens by whitespace, keeping all special characters.
Start and End tokens will be included.
It can return an empty list.
*/
func WhitespaceTokenize(s string) (tokens []Token){
    tokens = make([]Token, 0, 30) // guess as to typical length
    tokens = append(tokens, Token{isStart:true})
    for _, strTok := range strings.Fields(s){
        tokens = append(tokens, Token{value:strTok})
    }
    tokens = append(tokens, Token{isEnd:true})

    return
}

type Suffixes struct {
    // This double-storage allows fast retrieval of a random element.
    occurrences map[Token] float32 // suffix -> occurrence. Need to coerce to float32 when choosing, may as well save the cost then
    suffixes []Token // suffix1, suffix2, ...
}
func NewSuffixes() *Suffixes {
    return &Suffixes{occurrences: make(map[Token]float32)}
}

// Increment the count for this suffix.
func (s *Suffixes) Update(suffix Token){
    _, present := s.occurrences[suffix]
    if !present{
        s.suffixes = append(s.suffixes, suffix)
    }

    s.occurrences[suffix] += 1
}


// getRandom retrieves a random suffix, roughly proportional to its occurrence.
// O(1)
func (s *Suffixes) getRandom() Token{
    suffixes := [NumSamples]Token{}
    occurrences := [NumSamples]float32{}
    numSuf := len(s.suffixes)

    // select N elements in a row
    // then choose among those proportional to their occurrence.
    // unfair if len(suffixes) < NGrams.

    startIndex := rand.Intn(numSuf)
    var occursSum float32 = 0
    for i := 0; i < NGrams; i++{
        realIndex := (startIndex + i) % numSuf
        choice := s.suffixes[realIndex]
        suffixes[i] = choice

        occurs := s.occurrences[choice]
        occurrences[i] = occurs
        occursSum += occurs
    }

    return weightedChoice(suffixes, occurrences, rand.Float32())
}

// weightedChoice returns an elements of suffixes, weighted by occurs and chosen by r in [0.0, 1.0).
func weightedChoice(suffixes [NumSamples]Token, occurrences [NumSamples]float32, r float32) Token{
    // split elements into NumSamples buckets along [0, 1], then use r to pick one.

    var sum float32 = 0
    for _, e := range occurrences {
        sum += float32(e) // not worrying about overflow; shouldn't have such huge occurences
    }

    boundaries := make([]float32, NumSamples)
    var runningTotal float32 = 0
    for i := 0; i < NumSamples; i++ {
        runningTotal += float32(occurrences[i])
        boundaries[i] = runningTotal / sum
    }

    choice := 0
    for boundaries[choice] <= r && choice < NumSamples {
        choice += 1
    }

    return suffixes[choice]
}
