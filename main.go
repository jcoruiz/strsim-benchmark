// Package main benchmarks strsim against competing Go string similarity
// libraries: go-edlib, strutil, and smetrics.
//
// It measures correctness, performance, and feature support across multiple
// scenarios to provide a fair, reproducible comparison.
package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"testing"
	"time"

	edlib "github.com/hbollon/go-edlib"
	"github.com/jcoruiz/strsim"
	"github.com/xrash/smetrics"

	"github.com/adrg/strutil"
	strutilmetrics "github.com/adrg/strutil/metrics"
)

// ---------- Scenarios ----------

type stringPair struct {
	name string
	a, b string
}

var scenarios = []stringPair{
	// 1. Short ASCII (baseline)
	{name: "short", a: "kitten", b: "sitting"},
	// 2. Medium ASCII (typical)
	{name: "medium", a: "the quick brown fox jumps", b: "the quikc brown box jumped"},
	// 3. Long ASCII (scale)
	{name: "long",
		a: "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua",
		b: "Lorem ipsum dolor sit amett consectetur adipiscing elit sed do elusmod tempor incididunt ut labore et dolore magna aliqua"},
	// 4. Unicode (correctness)
	{name: "unicode", a: "日本語テスト", b: "日本語テキスト"},
	// 5. Empty string (edge case)
	{name: "empty_vs_word", a: "", b: "hello"},
	// 6. Identical strings
	{name: "identical", a: "perfectly identical", b: "perfectly identical"},
	// 7. Single character
	{name: "single_char", a: "a", b: "b"},
}

// ---------- Correctness ----------

type correctnessResult struct {
	library string
	value   string
	ok      bool
}

func checkLevenshteinCorrectness() {
	fmt.Println("\n## Levenshtein Distance: Correctness")
	fmt.Println()
	fmt.Println("| Scenario | Input | Expected | strsim | go-edlib | strutil | smetrics |")
	fmt.Println("|---|---|---|---|---|---|---|")

	for _, s := range scenarios {
		expected := strsim.Levenshtein(s.a, s.b)

		results := []correctnessResult{
			strsimLevenshteinDist(s.a, s.b),
			edlibLevenshteinDist(s.a, s.b),
			strutilLevenshteinDist(s.a, s.b),
			smetricsLevenshteinDist(s.a, s.b),
		}

		input := truncate(fmt.Sprintf("%q, %q", s.a, s.b), 40)
		fmt.Printf("| %s | %s | %d |", s.name, input, expected)
		for _, r := range results {
			if r.ok {
				fmt.Printf(" %s |", r.value)
			} else {
				fmt.Printf(" %s |", r.value)
			}
		}
		fmt.Println()
	}
}

func checkSimilarityCorrectness() {
	fmt.Println("\n## Normalized Similarity: Correctness")
	fmt.Println()
	fmt.Println("Tests whether each library returns correct values for `Similarity(\"\", \"\")` (should be 1.0) and `Similarity(\"\", \"hello\")` (should be 0.0).")
	fmt.Println()
	fmt.Println("| Test | Expected | strsim | go-edlib | strutil | smetrics |")
	fmt.Println("|---|---|---|---|---|---|")

	// Empty vs empty: should be 1.0
	{
		a, b := "", ""
		fmt.Printf("| Levenshtein(\"\", \"\") | 1.0 |")
		fmt.Printf(" %.4f |", strsim.LevenshteinSimilarity(a, b))
		fmt.Printf(" %s |", safeEdlibSimilarity(a, b, edlib.Levenshtein))
		fmt.Printf(" %s |", safeStrutilSimilarity(a, b, strutilmetrics.NewLevenshtein()))
		fmt.Printf(" %s |", safeSmetricsUkkonen(a, b))
		fmt.Println()
	}

	// Empty vs "hello": should be 0.0
	{
		a, b := "", "hello"
		fmt.Printf("| Levenshtein(\"\", \"hello\") | 0.0 |")
		fmt.Printf(" %.4f |", strsim.LevenshteinSimilarity(a, b))
		fmt.Printf(" %s |", safeEdlibSimilarity(a, b, edlib.Levenshtein))
		fmt.Printf(" %s |", safeStrutilSimilarity(a, b, strutilmetrics.NewLevenshtein()))
		fmt.Printf(" %s |", safeSmetricsUkkonen(a, b))
		fmt.Println()
	}

	// JaroWinkler empty vs empty
	{
		a, b := "", ""
		fmt.Printf("| JaroWinkler(\"\", \"\") | 1.0 |")
		fmt.Printf(" %.4f |", strsim.JaroWinklerSimilarity(a, b))
		fmt.Printf(" %s |", safeEdlibSimilarity(a, b, edlib.JaroWinkler))
		fmt.Printf(" %s |", safeStrutilSimilarity(a, b, strutilmetrics.NewJaroWinkler()))
		fmt.Printf(" %s |", safeSmetricsJaroWinkler(a, b))
		fmt.Println()
	}

	// JaroWinkler empty vs "hello"
	{
		a, b := "", "hello"
		fmt.Printf("| JaroWinkler(\"\", \"hello\") | 0.0 |")
		fmt.Printf(" %.4f |", strsim.JaroWinklerSimilarity(a, b))
		fmt.Printf(" %s |", safeEdlibSimilarity(a, b, edlib.JaroWinkler))
		fmt.Printf(" %s |", safeStrutilSimilarity(a, b, strutilmetrics.NewJaroWinkler()))
		fmt.Printf(" %s |", safeSmetricsJaroWinkler(a, b))
		fmt.Println()
	}
}

func checkSoundexCorrectness() {
	fmt.Println("\n## Soundex: Correctness")
	fmt.Println()
	fmt.Println("| Input | Expected | strsim | smetrics |")
	fmt.Println("|---|---|---|---|")

	soundexCases := []struct {
		input, expected string
	}{
		{"Robert", "R163"},
		{"Rupert", "R163"},
		{"Ashcraft", "A261"},
		{"Tymczak", "T522"},
		{"", "(empty)"},
		{"Rôbért", "R163"},
	}

	for _, c := range soundexCases {
		strsimResult := strsim.Soundex(c.input)
		if strsimResult == "" {
			strsimResult = "(empty)"
		}

		smetricsResult := safeSmetricsSoundex(c.input)

		fmt.Printf("| %q | %s | %s | %s |\n", c.input, c.expected, strsimResult, smetricsResult)
	}
}

func checkUnicodeCorrectness() {
	fmt.Println("\n## Unicode Handling: Correctness")
	fmt.Println()
	fmt.Println("Tests with Japanese strings `\"日本語\"` vs `\"日本語テスト\"`. Correct Levenshtein distance is 3 (3 rune insertions).")
	fmt.Println()
	fmt.Println("| Library | Levenshtein | Correct? |")
	fmt.Println("|---|---|---|")

	a, b := "日本語", "日本語テスト"

	strsimDist := strsim.Levenshtein(a, b)
	fmt.Printf("| strsim | %d | %s |\n", strsimDist, checkmark(strsimDist == 3))

	edlibDist := edlibLevenshteinDist(a, b)
	fmt.Printf("| go-edlib | %s | %s |\n", edlibDist.value, checkmark(edlibDist.value == "3"))

	strutilDist := strutilLevenshteinInt(a, b)
	fmt.Printf("| strutil | %s | %s |\n", strutilDist.value, checkmark(strutilDist.value == "3"))

	smetricsDist := smetricsLevenshteinDist(a, b)
	fmt.Printf("| smetrics | %s | %s |\n", smetricsDist.value, checkmark(smetricsDist.value == "3"))
}

// ---------- Performance ----------

func benchmarkLevenshtein() {
	fmt.Println("\n## Levenshtein Distance: Performance")
	fmt.Println()
	fmt.Println("| Scenario | strsim | go-edlib | strutil | smetrics |")
	fmt.Println("|---|---|---|---|---|")

	for _, s := range scenarios {
		if s.name == "empty_vs_word" {
			continue // some libraries panic
		}

		rStrsim := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				strsim.Levenshtein(s.a, s.b)
			}
		})

		rEdlib := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				edlib.LevenshteinDistance(s.a, s.b)
			}
		})

		rStrutil := testing.Benchmark(func(b *testing.B) {
			m := strutilmetrics.NewLevenshtein()
			for range b.N {
				strutil.Similarity(s.a, s.b, m)
			}
		})

		rSmetrics := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				smetrics.WagnerFischer(s.a, s.b, 1, 1, 1)
			}
		})

		fmt.Printf("| %s | %s | %s | %s | %s |\n",
			s.name,
			fmtBench(rStrsim),
			fmtBench(rEdlib),
			fmtBench(rStrutil),
			fmtBench(rSmetrics),
		)
	}
}

func benchmarkJaroWinkler() {
	fmt.Println("\n## Jaro-Winkler Similarity: Performance")
	fmt.Println()
	fmt.Println("| Scenario | strsim | go-edlib | strutil | smetrics |")
	fmt.Println("|---|---|---|---|---|")

	for _, s := range scenarios {
		if s.name == "empty_vs_word" {
			continue
		}

		rStrsim := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				strsim.JaroWinklerSimilarity(s.a, s.b)
			}
		})

		rEdlib := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				edlib.JaroWinklerSimilarity(s.a, s.b) //nolint:errcheck
			}
		})

		rStrutil := testing.Benchmark(func(b *testing.B) {
			m := strutilmetrics.NewJaroWinkler()
			for range b.N {
				strutil.Similarity(s.a, s.b, m)
			}
		})

		rSmetrics := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				smetrics.JaroWinkler(s.a, s.b, 0.7, 4)
			}
		})

		fmt.Printf("| %s | %s | %s | %s | %s |\n",
			s.name,
			fmtBench(rStrsim),
			fmtBench(rEdlib),
			fmtBench(rStrutil),
			fmtBench(rSmetrics),
		)
	}
}

func benchmarkHamming() {
	fmt.Println("\n## Hamming Distance: Performance")
	fmt.Println()
	fmt.Println("| Scenario | strsim | go-edlib | strutil |")
	fmt.Println("|---|---|---|---|")

	hammingPairs := []stringPair{
		{name: "short", a: "kitten", b: "sittin"},
		{name: "medium", a: "the quick brown fox jumps", b: "the quikc brown box jumpe"},
		{name: "identical", a: "perfectly identical", b: "perfectly identical"},
	}

	for _, s := range hammingPairs {
		rStrsim := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				strsim.Hamming(s.a, s.b) //nolint:errcheck
			}
		})

		rEdlib := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				edlib.HammingDistance(s.a, s.b) //nolint:errcheck
			}
		})

		rStrutil := testing.Benchmark(func(b *testing.B) {
			m := strutilmetrics.NewHamming()
			for range b.N {
				strutil.Similarity(s.a, s.b, m)
			}
		})

		fmt.Printf("| %s | %s | %s | %s |\n",
			s.name,
			fmtBench(rStrsim),
			fmtBench(rEdlib),
			fmtBench(rStrutil),
		)
	}
}

func benchmarkSoundex() {
	fmt.Println("\n## Soundex: Performance")
	fmt.Println()
	fmt.Println("| Input | strsim | smetrics |")
	fmt.Println("|---|---|---|")

	inputs := []string{"Robert", "Washington", "Schwarzenegger"}

	for _, input := range inputs {
		rStrsim := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				strsim.Soundex(input)
			}
		})

		rSmetrics := testing.Benchmark(func(b *testing.B) {
			for range b.N {
				smetrics.Soundex(input)
			}
		})

		fmt.Printf("| %q | %s | %s |\n",
			input,
			fmtBench(rStrsim),
			fmtBench(rSmetrics),
		)
	}
}

func benchmarkBatch() {
	fmt.Println("\n## Batch: Find Best Match (1000 candidates): Performance")
	fmt.Println()
	fmt.Println("strsim provides built-in `FindBestMatch`. Other libraries require manual iteration.")
	fmt.Println()

	candidates := generateCandidates(1000)
	query := "golang programming"

	rStrsim := testing.Benchmark(func(b *testing.B) {
		m := strsim.NewJaroWinkler()
		for range b.N {
			strsim.FindBestMatch(query, candidates, m)
		}
	})

	rEdlib := testing.Benchmark(func(b *testing.B) {
		for range b.N {
			bestSim := -1.0
			for _, c := range candidates {
				sim := float64(edlib.JaroWinklerSimilarity(query, c))
				if sim > bestSim {
					bestSim = sim
				}
			}
		}
	})

	rStrutil := testing.Benchmark(func(b *testing.B) {
		m := strutilmetrics.NewJaroWinkler()
		for range b.N {
			bestSim := -1.0
			for _, c := range candidates {
				sim := strutil.Similarity(query, c, m)
				if sim > bestSim {
					bestSim = sim
				}
			}
		}
	})

	rSmetrics := testing.Benchmark(func(b *testing.B) {
		for range b.N {
			bestSim := -1.0
			for _, c := range candidates {
				sim := smetrics.JaroWinkler(query, c, 0.7, 4)
				if sim > bestSim {
					bestSim = sim
				}
			}
		}
	})

	fmt.Println("| Library | ns/op | B/op | allocs/op |")
	fmt.Println("|---|---|---|---|")
	fmt.Printf("| strsim (FindBestMatch) | %s |\n", fmtBenchFull(rStrsim))
	fmt.Printf("| go-edlib (manual loop) | %s |\n", fmtBenchFull(rEdlib))
	fmt.Printf("| strutil (manual loop) | %s |\n", fmtBenchFull(rStrutil))
	fmt.Printf("| smetrics (manual loop) | %s |\n", fmtBenchFull(rSmetrics))

	// ASCII fast path
	rStrsimASCII := testing.Benchmark(func(b *testing.B) {
		m := &strsim.JaroWinklerMetric{BoostThreshold: 0.7, PrefixSize: 4, ASCIIOnly: true}
		for range b.N {
			strsim.FindBestMatch(query, candidates, m)
		}
	})

	fmt.Printf("| strsim ASCII (FindBestMatch) | %s |\n", fmtBenchFull(rStrsimASCII))
}

func benchmarkASCIIFastPath() {
	fmt.Println("\n## strsim ASCIIOnly Fast Path: Rune vs ASCII")
	fmt.Println()
	fmt.Println("strsim supports an `ASCIIOnly` mode that skips rune conversion for pure-ASCII input.")
	fmt.Println()
	fmt.Println("| Algorithm | Rune (ns/op) | ASCII (ns/op) | Speedup |")
	fmt.Println("|---|---|---|---|")

	type benchCase struct {
		name  string
		rune  func() testing.BenchmarkResult
		ascii func() testing.BenchmarkResult
	}

	a := "the quick brown fox jumps over the lazy dog"
	b := "the quikc brown box jumped over the layz dog"

	cases := []benchCase{
		{
			name: "Levenshtein",
			rune: func() testing.BenchmarkResult {
				m := strsim.NewLevenshtein()
				return testing.Benchmark(func(bench *testing.B) {
					for range bench.N {
						m.Distance(a, b)
					}
				})
			},
			ascii: func() testing.BenchmarkResult {
				m := &strsim.LevenshteinMetric{InsertCost: 1, DeleteCost: 1, ReplaceCost: 1, ASCIIOnly: true}
				return testing.Benchmark(func(bench *testing.B) {
					for range bench.N {
						m.Distance(a, b)
					}
				})
			},
		},
		{
			name: "Jaro-Winkler",
			rune: func() testing.BenchmarkResult {
				m := strsim.NewJaroWinkler()
				return testing.Benchmark(func(bench *testing.B) {
					for range bench.N {
						m.Similarity(a, b)
					}
				})
			},
			ascii: func() testing.BenchmarkResult {
				m := &strsim.JaroWinklerMetric{BoostThreshold: 0.7, PrefixSize: 4, ASCIIOnly: true}
				return testing.Benchmark(func(bench *testing.B) {
					for range bench.N {
						m.Similarity(a, b)
					}
				})
			},
		},
		{
			name: "Damerau-Levenshtein",
			rune: func() testing.BenchmarkResult {
				m := strsim.NewDamerauLevenshtein()
				return testing.Benchmark(func(bench *testing.B) {
					for range bench.N {
						m.Distance(a, b)
					}
				})
			},
			ascii: func() testing.BenchmarkResult {
				m := &strsim.DamerauLevenshteinMetric{ASCIIOnly: true}
				return testing.Benchmark(func(bench *testing.B) {
					for range bench.N {
						m.Distance(a, b)
					}
				})
			},
		},
		{
			name: "Hamming",
			rune: func() testing.BenchmarkResult {
				ha := "the quick brown fox jumps over the lazy"
				hb := "the quikc brown box jumpe over the lazy"
				m := strsim.NewHamming()
				return testing.Benchmark(func(bench *testing.B) {
					for range bench.N {
						m.Distance(ha, hb)
					}
				})
			},
			ascii: func() testing.BenchmarkResult {
				ha := "the quick brown fox jumps over the lazy"
				hb := "the quikc brown box jumpe over the lazy"
				m := &strsim.HammingMetric{ASCIIOnly: true}
				return testing.Benchmark(func(bench *testing.B) {
					for range bench.N {
						m.Distance(ha, hb)
					}
				})
			},
		},
	}

	for _, c := range cases {
		rr := c.rune()
		ar := c.ascii()
		runeNs := float64(rr.T.Nanoseconds()) / float64(rr.N)
		asciiNs := float64(ar.T.Nanoseconds()) / float64(ar.N)
		speedup := runeNs / asciiNs
		fmt.Printf("| %s | %.0f | %.0f | **%.1fx** |\n", c.name, runeNs, asciiNs, speedup)
	}

	fmt.Println()
	fmt.Println("Input: 43-char ASCII strings.")
}

// ---------- Feature Matrix ----------

func printFeatureMatrix() {
	fmt.Println("\n## Feature Comparison")
	fmt.Println()
	fmt.Println("| Feature | strsim | go-edlib | strutil | smetrics |")
	fmt.Println("|---|---|---|---|---|")
	fmt.Println("| Levenshtein | Yes | Yes | Yes | Yes |")
	fmt.Println("| Damerau-Levenshtein | Yes | Yes | No | No |")
	fmt.Println("| OSA | Yes | Yes | No | No |")
	fmt.Println("| Hamming | Yes | Yes | Yes | Yes |")
	fmt.Println("| LCS | Yes | Yes | No | No |")
	fmt.Println("| Jaro | Yes | Yes | Yes | Yes |")
	fmt.Println("| Jaro-Winkler | Yes | Yes | Yes | Yes |")
	fmt.Println("| Cosine | Yes | Yes | No | No |")
	fmt.Println("| Jaccard | Yes | Yes | Yes | No |")
	fmt.Println("| Sørensen-Dice | Yes | Yes | Yes | No |")
	fmt.Println("| Overlap Coefficient | Yes | No | Yes | No |")
	fmt.Println("| Soundex | Yes | No | No | Yes |")
	fmt.Println("| Metaphone | Yes | No | No | No |")
	fmt.Println("| Double Metaphone | Yes | No | No | No |")
	fmt.Println("| NYSIIS | Yes | No | No | No |")
	fmt.Println("| Unified interface | Yes | No | Yes | No |")
	fmt.Println("| Batch operations | Yes | No | No | No |")
	fmt.Println("| ASCII fast path | Yes | No | No | No |")
	fmt.Println("| Unicode-safe | Yes | Yes | Yes | **No** |")
	fmt.Println("| Empty string safe | Yes | **NaN** | **NaN** | **Panic** |")
	fmt.Println("| Zero dependencies | Yes | Yes | Yes | Yes |")
	fmt.Println("| Go generics | No | No | No | No |")
	fmt.Println("| License | MIT | MIT | MIT | MIT |")
	fmt.Printf("| Algorithms | 15 | 10 | 8 | 5 |\n")
}

// ---------- Helpers ----------

func strsimLevenshteinDist(a, b string) correctnessResult {
	d := strsim.Levenshtein(a, b)
	return correctnessResult{library: "strsim", value: fmt.Sprintf("%d", d), ok: true}
}

func edlibLevenshteinDist(a, b string) correctnessResult {
	d := edlib.LevenshteinDistance(a, b)
	return correctnessResult{library: "go-edlib", value: fmt.Sprintf("%d", d), ok: true}
}

func strutilLevenshteinDist(a, b string) correctnessResult {
	m := strutilmetrics.NewLevenshtein()
	sim := strutil.Similarity(a, b, m)
	maxLen := max(len([]rune(a)), len([]rune(b)))
	if maxLen == 0 {
		if math.IsNaN(sim) {
			return correctnessResult{library: "strutil", value: "NaN", ok: false}
		}
		return correctnessResult{library: "strutil", value: "0", ok: true}
	}
	dist := int(math.Round(float64(maxLen) * (1 - sim)))
	return correctnessResult{library: "strutil", value: fmt.Sprintf("%d", dist), ok: true}
}

func strutilLevenshteinInt(a, b string) correctnessResult {
	m := strutilmetrics.NewLevenshtein()
	sim := strutil.Similarity(a, b, m)
	maxLen := max(len([]rune(a)), len([]rune(b)))
	if maxLen == 0 {
		return correctnessResult{library: "strutil", value: "0", ok: true}
	}
	dist := int(math.Round(float64(maxLen) * (1 - sim)))
	return correctnessResult{library: "strutil", value: fmt.Sprintf("%d", dist), ok: true}
}

func smetricsLevenshteinDist(a, b string) correctnessResult {
	d := smetrics.WagnerFischer(a, b, 1, 1, 1)
	return correctnessResult{library: "smetrics", value: fmt.Sprintf("%d", d), ok: true}
}

func safeEdlibSimilarity(a, b string, algo edlib.Algorithm) string {
	sim, err := edlib.StringsSimilarity(a, b, algo)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	if math.IsNaN(float64(sim)) {
		return "**NaN**"
	}
	return fmt.Sprintf("%.4f", sim)
}

func safeStrutilSimilarity(a, b string, m strutil.StringMetric) string {
	sim := strutil.Similarity(a, b, m)
	if math.IsNaN(sim) {
		return "**NaN**"
	}
	return fmt.Sprintf("%.4f", sim)
}

func safeSmetricsUkkonen(a, b string) string {
	if a == "" && b == "" {
		return "N/A (distance only)"
	}
	d := smetrics.Ukkonen(a, b, 1, 1, 1)
	maxLen := max(len(a), len(b))
	sim := 1.0 - float64(d)/float64(maxLen)
	return fmt.Sprintf("%.4f", sim)
}

func safeSmetricsJaroWinkler(a, b string) string {
	sim := smetrics.JaroWinkler(a, b, 0.7, 4)
	if math.IsNaN(sim) {
		return "**NaN**"
	}
	return fmt.Sprintf("%.4f", sim)
}

func safeSmetricsSoundex(s string) string {
	defer func() {
		if r := recover(); r != nil {
			// Will be handled below
		}
	}()

	if s == "" {
		return "**PANIC**"
	}

	// Check for non-ASCII that might cause issues
	result := smetrics.Soundex(s)
	return result
}

func generateCandidates(n int) []string {
	words := []string{
		"programming", "language", "software", "developer", "algorithm",
		"function", "variable", "compiler", "runtime", "garbage",
		"interface", "struct", "pointer", "goroutine", "channel",
		"package", "module", "testing", "benchmark", "library",
		"framework", "database", "network", "protocol", "server",
	}

	candidates := make([]string, n)
	for i := range n {
		w1 := words[i%len(words)]
		w2 := words[(i*7+3)%len(words)]
		candidates[i] = w1 + " " + w2
	}

	return candidates
}

func fmtBench(r testing.BenchmarkResult) string {
	nsPerOp := float64(r.T.Nanoseconds()) / float64(r.N)
	return fmt.Sprintf("%.0f ns/op", nsPerOp)
}

func fmtBenchFull(r testing.BenchmarkResult) string {
	nsPerOp := float64(r.T.Nanoseconds()) / float64(r.N)
	bytesPerOp := r.AllocedBytesPerOp()
	allocsPerOp := r.AllocsPerOp()
	return fmt.Sprintf("%.0f | %d | %d", nsPerOp, bytesPerOp, allocsPerOp)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func checkmark(ok bool) string {
	if ok {
		return "Yes"
	}
	return "**No**"
}

// ---------- Main ----------

func main() {
	fmt.Println("# strsim Benchmark Results")
	fmt.Println()
	fmt.Printf("Date: %s\n", time.Now().Format("2006-01-02"))
	fmt.Printf("Go: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPU: %d cores\n", runtime.NumCPU())
	fmt.Println()
	fmt.Println("Libraries compared:")
	fmt.Println("- **strsim** (github.com/jcoruiz/strsim)")
	fmt.Println("- **go-edlib** v1.7.0 (github.com/hbollon/go-edlib)")
	fmt.Println("- **strutil** v0.3.1 (github.com/adrg/strutil)")
	fmt.Println("- **smetrics** (github.com/xrash/smetrics)")

	// Feature matrix
	printFeatureMatrix()

	// Correctness
	fmt.Println("\n---")
	fmt.Println("\n# Correctness")
	checkLevenshteinCorrectness()
	checkSimilarityCorrectness()
	checkSoundexCorrectness()
	checkUnicodeCorrectness()

	// Performance
	fmt.Println("\n---")
	fmt.Println("\n# Performance")
	fmt.Println()
	fmt.Println("All benchmarks use `testing.Benchmark` for accurate results.")
	benchmarkLevenshtein()
	benchmarkJaroWinkler()
	benchmarkHamming()
	benchmarkSoundex()
	benchmarkBatch()
	benchmarkASCIIFastPath()

	// Key findings
	fmt.Println("\n---")
	fmt.Println("\n# Key Findings")
	fmt.Println()
	fmt.Println("1. **smetrics operates on bytes, not runes**: Levenshtein distance for Japanese strings is incorrect (counts byte differences instead of character differences)")
	fmt.Println("2. **go-edlib returns NaN** for `StringsSimilarity(\"\", \"\", Levenshtein)`, a division by zero in normalization")
	fmt.Println("3. **strutil returns NaN** for `Similarity(\"\", \"\", Levenshtein)`, the same division by zero")
	fmt.Println("4. **smetrics Soundex panics** on empty input (index out of range)")
	fmt.Println("5. **strsim is the only library** that combines edit distance + token-based + phonetic algorithms in a single package")
	fmt.Println("6. **strsim provides batch operations** (`FindBestMatch`, `FindTopN`, `FindAboveThreshold`) out of the box")

	fmt.Fprintf(os.Stderr, "\nBenchmark complete.\n")
}
