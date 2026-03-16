# strsim-benchmark

Reproducible benchmark comparing Go string similarity libraries.

## Libraries

| Library | Version | Stars | Algorithms | License |
|---|---|---|---|---|
| [strsim](https://github.com/jcoruiz/strsim) | v0.1.0 | - | 15 (edit + token + phonetic) | MIT |
| [go-edlib](https://github.com/hbollon/go-edlib) | v1.7.0 | 595 | 10 (edit + token) | MIT |
| [strutil](https://github.com/adrg/strutil) | v0.3.1 | 415 | 8 (edit + token) | MIT |
| [smetrics](https://github.com/xrash/smetrics) | latest | 236 | 5 (edit) + Soundex | MIT |

## Running

```bash
go run .
```

Requires Go 1.22+. The benchmark takes about 30 seconds.

## Results

Date: 2026-03-16 | Go 1.22.2 | Linux/amd64 | 32 cores

### Feature Comparison

| Feature | strsim | go-edlib | strutil | smetrics |
|---|---|---|---|---|
| Levenshtein | Yes | Yes | Yes | Yes |
| Damerau-Levenshtein | Yes | Yes | No | No |
| OSA | Yes | Yes | No | No |
| Hamming | Yes | Yes | Yes | Yes |
| LCS | Yes | Yes | No | No |
| Jaro | Yes | Yes | Yes | Yes |
| Jaro-Winkler | Yes | Yes | Yes | Yes |
| Cosine | Yes | Yes | No | No |
| Jaccard | Yes | Yes | Yes | No |
| Sørensen-Dice | Yes | Yes | Yes | No |
| Overlap Coefficient | Yes | No | Yes | No |
| Soundex | Yes | No | No | Yes |
| Metaphone | Yes | No | No | No |
| Double Metaphone | Yes | No | No | No |
| NYSIIS | Yes | No | No | No |
| Unified interface | Yes | No | Yes | No |
| Batch operations | Yes | No | No | No |
| ASCII fast path | Yes | No | No | No |
| Unicode-safe | Yes | Yes | Yes | **No** |
| Empty string safe | Yes | **NaN** | **NaN** | **Panic** |
| Total algorithms | **15** | 10 | 8 | 5 |

### Correctness: Edge Cases

| Test | Expected | strsim | go-edlib | strutil | smetrics |
|---|---|---|---|---|---|
| Levenshtein("", "") similarity | 1.0 | 1.0 | **NaN** | **NaN** | N/A |
| JaroWinkler("", "") similarity | 1.0 | 1.0 | 0.0 | 1.0 | 1.0 |
| Soundex("") | (empty) | (empty) | N/A | N/A | **PANIC** |
| Soundex("Ashcraft") | A261 | A261 | N/A | N/A | **A226** |
| Levenshtein("日本語", "日本語テスト") | 3 | 3 | 3 | 3 | **9** (bytes) |

### Performance: Levenshtein Distance (ns/op)

| Scenario | strsim | go-edlib | strutil | smetrics |
|---|---|---|---|---|
| Short (7 chars) | 90 | 69 | 114 | 69 |
| Medium (25 chars) | 977 | 1,030 | 1,759 | 598* |
| Long (117 chars) | 19,947 | 20,094 | 39,300 | 11,986* |
| Unicode | 129 | 110 | 153 | 344 |
| Identical | 581 | 34 | 1,034 | 326 |
| Single char | 27 | 17 | 27 | 21 |

\* smetrics is faster on long ASCII strings because it operates on bytes instead of runes, but this produces **incorrect results for Unicode input**.

### Performance: Jaro-Winkler Similarity (ns/op)

| Scenario | strsim | go-edlib | strutil | smetrics |
|---|---|---|---|---|
| Short (7 chars) | 52 | 72 | 129 | 49 |
| Medium (25 chars) | **189** | 272 | 531 | 210 |
| Long (117 chars) | **2,020** | 3,117 | 4,511 | 2,697 |
| Unicode | **128** | 146 | 307 | 136 |
| Identical | 134 | 34 | 464 | 142 |

### Performance: Batch Find Best Match (1000 candidates, Jaro-Winkler)

| Library | ns/op | B/op | allocs/op |
|---|---|---|---|
| strsim (FindBestMatch) | 166,372 | 43,520 | 2,000 |
| go-edlib (manual loop) | 214,263 | 280,322 | 2,000 |
| strutil (manual loop) | 347,563 | 168,321 | 6,880 |
| smetrics (manual loop) | 157,323 | 43,520 | 2,000 |

### Performance: Soundex (ns/op)

| Input | strsim | smetrics |
|---|---|---|
| "Robert" | 25 | 15 |
| "Washington" | 30 | 17 |
| "Schwarzenegger" | 38 | 16 |

smetrics Soundex is faster but has bugs: **panics on empty input** and produces **incorrect results for Ashcraft** (A226 instead of A261 — doesn't handle H/W transparency correctly per the American Soundex specification).

### strsim ASCIIOnly Fast Path

strsim supports an `ASCIIOnly` mode that skips rune conversion for pure-ASCII input. This trades Unicode correctness for speed — use only when you know your input is ASCII.

| Algorithm | Rune (ns/op) | ASCII (ns/op) | Speedup |
|---|---|---|---|
| Hamming | 116 | 10 | **11.3x** |
| Damerau-Levenshtein | 17,250 | 7,019 | **2.5x** |
| Jaro-Winkler | 484 | 224 | **2.2x** |
| Levenshtein | 3,016 | 2,806 | 1.1x |

Input: 43-char ASCII strings. With `ASCIIOnly`, strsim Levenshtein matches smetrics speed while being explicit about the ASCII-only trade-off (smetrics always operates on bytes with no opt-in).

## Key Findings

1. **smetrics operates on bytes, not runes** — produces incorrect Levenshtein distance for non-ASCII input (9 instead of 3 for Japanese strings)
2. **go-edlib and strutil return NaN** for `Similarity("", "")` — division by zero in normalization
3. **smetrics Soundex panics** on empty input and gives wrong results for "Ashcraft" (H/W transparency bug)
4. **strsim Jaro-Winkler is fastest** for medium and long strings among Unicode-correct libraries
5. **strsim is the only library** combining edit distance + token-based + phonetic (15 algorithms)
6. **strsim provides batch operations** (`FindBestMatch`, `FindTopN`, `FindAboveThreshold`) out of the box
7. **strsim ASCIIOnly mode** gives up to 11x speedup for ASCII input, making the byte-vs-rune trade-off explicit (unlike smetrics which always uses bytes silently)

## Methodology

- All benchmarks use Go's `testing.Benchmark` for accurate results
- Same inputs across all libraries per scenario
- Deterministic inputs (no randomness)
- Unicode correctness verified by comparing rune-level vs byte-level distance
- Empty string handling tested explicitly
- Run on the same machine in a single invocation to minimize variance

## Reproducing

```bash
git clone https://github.com/jcoruiz/strsim-benchmark.git
cd strsim-benchmark
go run .
```
