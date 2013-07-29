package markov

import (
    "fmt"
	"testing"
    //"github.com/davecgh/go-spew/spew"
    "strings"
)

func getTestTokens() map[string]Token{
    // it's a pita to make tokens on the fly
    tokens := make(map[string]Token)
    for _, letter := range strings.Fields("a b c d"){
        tokens[letter] = *NewToken(letter)
    }
    tokens["start"] = Token{isStart:true}
    tokens["end"] = Token{isEnd:true}

    return tokens
}

func TestRespond(t *testing.T){
    c := NewChain()
    c.Update("a b c")
    c.Update("b c d")
    c.Update("c d")

    abc, err := c.Respond("a b", 2, 3)
    if err != nil{
        t.Fatal(err)
    }
    if abc != "a b c"{
        fmt.Println(abc)
        t.Error("expected \"a b c\"")
    }
}

func Testrespond(t *testing.T){
    c := NewChain()
    c.Update("a b c")
    c.Update("b c d")
    c.Update("c d")

    tokens := getTestTokens()

    ab := [2]Token{tokens["a"], tokens["b"]}
    emptyResponse := c.respond(ab, 0)
    if len(emptyResponse) != 0{
        t.Error("expected empty response")
    }
}

func TestLearning(t *testing.T){
    c := NewChain()
    c.Update("a b c")
    c.Update("b c d")
    c.Update("c d")

    tokens := getTestTokens()

    ab := [2]Token{tokens["a"], tokens["b"]}

    if len(c.chains[ab].suffixes) != 1{
        t.Error("ab should have 1 suffix")
    }
    if c.chains[ab].suffixes[0] != tokens["c"]{
        t.Error("ab->c should occur")
    }
    if c.chains[ab].occurrences[tokens["c"]] != 1{
        t.Error("ab->c should occur once")
    }

    bc := [2]Token{tokens["b"], tokens["c"]}
    if len(c.chains[bc].suffixes) != 2{
        t.Error("bc should have 2 suffixes")
    }
    if c.chains[bc].occurrences[tokens["d"]] != 1{
        t.Error("bc->d should occur once")
    }
    if c.chains[bc].occurrences[tokens["end"]] != 1{
        t.Error("bc->end should occur once")
    }
}

func TestWhitespaceTokenize(t *testing.T){
    tokens := WhitespaceTokenize("a! b/c D.")
    expected := [3]string{"a!", "b/c", "D."}

    start := Token{isStart: true}
    end := Token{isEnd: true}

    if tokens[0] != start{
        t.Error("expected start token")
    }
    if tokens[4] != end{
        t.Error("expected end token")
    }
    if len(tokens) != 5{
        t.Error("incorrect amount of tokens")
    }

    for i := 1; i < 4; i++{
        if tokens[i].value != expected[i-1]{
            t.Errorf("incorrect token in position %d", i)
        }
    }
}

func TestWeightedChoice(t *testing.T) {
    suffixes := [3]Token{*NewToken("a"), *NewToken("b"), *NewToken("c")}
    occurrences := [3]float32{2, 5, 1}

    if "a" != weightedChoice(suffixes, occurrences, 0).value{
        t.Error("0 should choose a")
    }
    if "b" != weightedChoice(suffixes, occurrences, .25).value {
        t.Error(".25 should choose b")
    }
    if "c" != weightedChoice(suffixes, occurrences, .99).value {
        t.Error(".99 should chooce c")
    }
}

func TestSuffixUpdate(t *testing.T){
    s := NewSuffixes()
    fooTok := *NewToken("foo");
    s.Update(fooTok)

    if len(s.suffixes) != 1 {
        t.Error("len(suffixes) should be 1")
    }
    if s.suffixes[0] != fooTok{
        t.Error("el should be present")
    }
    if s.occurrences[fooTok] != 1{
        t.Error("el occurencces should be 1")
    }

    s.Update(fooTok)
    if len(s.suffixes) != 1 {
        t.Error("len(suffixes) should be 1 after duplicate update")
    }
    if s.occurrences[fooTok] != 2{
        t.Error("el occurrences should be 2")
    }
}
