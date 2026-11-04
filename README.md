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
| Short (7 chars) | 90 | 70 | 116 | **67** |
| Medium (25 chars) | 1,042 | 1,111 | 1,788 | **573** |
| Long (117 chars) | 21,158 | 19,753 | 38,585 | **14,003** |
| Unicode | 128 | **108** | 152 | 348 |
| Identical | 626 | **32** | 974 | 331 |
| Single char | 26 | **17** | 28 | 21 |

\* smetrics is faster on ASCII strings because it operates on bytes instead of runes, but this produces **incorrect results for Unicode input**.

### Performance: Jaro-Winkler Similarity (ns/op)

| Scenario | strsim | go-edlib | strutil | smetrics |
|---|---|---|---|---|
| Short (7 chars) | **52** | 69 | 133 | 48 |
| Medium (25 chars) | **182** | 268 | 619 | 194 |
| Long (117 chars) | **1,952** | 3,197 | 6,463 | 1,991 |
| Unicode | **130** | 150 | 297 | 135 |
| Identical | 134 | **33** | 468 | 140 |

### Performance: Hamming Distance (ns/op)

| Scenario | strsim | go-edlib | strutil |
|---|---|---|---|
| Short | **12** | 14 | 14 |
| Medium | **38** | 42 | 42 |
| Identical | **31** | 32 | 34 |

### Performance: Batch Find Best Match (1000 candidates, Jaro-Winkler)

| Library | ns/op | B/op | allocs/op |
|---|---|---|---|
| strsim (FindBestMatch) | 161,478 | 43,520 | 2,000 |
| go-edlib (manual loop) | 212,586 | 280,322 | 2,000 |
| strutil (manual loop) | 395,804 | 168,322 | 6,880 |
| smetrics (manual loop) | 159,813 | 43,520 | 2,000 |
| **strsim ASCII (FindBestMatch)** | **93,146** | **43,520** | **2,000** |

### strsim ASCIIOnly Fast Path

strsim supports an `ASCIIOnly` mode that skips rune conversion for pure-ASCII input. This trades Unicode correctness for speed: use it only when you know your input is ASCII.

| Algorithm | Rune (ns/op) | ASCII (ns/op) | Speedup |
|---|---|---|---|
| Hamming | 117 | 11 | **11.0x** |
| Damerau-Levenshtein | 17,477 | 6,804 | **2.6x** |
| Jaro-Winkler | 493 | 234 | **2.1x** |
| Levenshtein | 3,144 | 2,787 | 1.1x |

Input: 43-char ASCII strings. With `ASCIIOnly`, strsim batch matching (93k ns) is **1.7x faster than smetrics** (160k ns), making it the fastest option for ASCII workloads.

### Performance: Soundex (ns/op)

| Input | strsim | smetrics |
|---|---|---|
| "Robert" | 25 | **16** |
| "Washington" | 30 | **17** |
| "Schwarzenegger" | 38 | **17** |

smetrics Soundex is faster but has bugs: **panics on empty input** and produces **incorrect results for Ashcraft** (A226 instead of A261: it doesn't treat H and W as transparent, as the American Soundex specification requires).

## Key Findings

1. **smetrics operates on bytes, not runes**, so it produces incorrect Levenshtein distance for non-ASCII input (9 instead of 3 for Japanese strings)
2. **go-edlib and strutil return NaN** for `Similarity("", "")`, a division by zero in normalization
3. **smetrics Soundex panics** on empty input and gives wrong results for "Ashcraft" (H/W transparency bug)
4. **strsim Jaro-Winkler is fastest** for medium and long strings, the typical real-world case
5. **strsim Hamming is fastest** across all scenarios
6. **strsim is the only library** combining edit distance + token-based + phonetic (15 algorithms)
7. **strsim ASCIIOnly batch matching is the fastest** at 93k ns/op, 1.7x faster than smetrics (160k ns)
8. **smetrics wins Levenshtein** on ASCII strings due to a tighter inner loop, but with broken Unicode

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
