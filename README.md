# strsim-benchmark

Reproducible benchmark comparing Go string similarity libraries.

## Libraries

| Library | Version | Stars | Algorithms | License |
|---|---|---|---|---|
| [strsim](https://github.com/jcoruiz/strsim) | v0.2.1 | - | 15 (edit + token + phonetic) | MIT |
| [go-edlib](https://github.com/hbollon/go-edlib) | v1.7.0 | 607 | 10 (edit + token) | MIT |
| [strutil](https://github.com/adrg/strutil) | v0.3.1 | 417 | 8 (edit + token) | MIT |
| [smetrics](https://github.com/xrash/smetrics) | 2025-07-05 commit | 235 | 5 (edit) + Soundex | MIT |

## Running

```bash
go run .
```

Requires Go 1.22+. The benchmark takes about two minutes.

## Results

Date: 2026-11-04 | Go 1.22.2 | Linux/amd64 | 32 cores | median of three runs, in ns/op unless noted

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
| Short (7 chars) | 88 | 70 | 110 | **66** |
| Medium (25 chars) | 945 | 1,018 | 1,806 | **565** |
| Long (117 chars) | 19,371 | 20,199 | 38,003 | **11,805** |
| Unicode | 124 | **106** | 148 | 333 |
| Identical | 555 | **33** | 985 | 317 |
| Single char | 26 | **17** | 27 | 21 |

\* smetrics is faster on ASCII strings because it operates on bytes instead of runes, but this produces **incorrect results for Unicode input**.

### Performance: Jaro-Winkler Similarity (ns/op)

| Scenario | strsim | go-edlib | strutil | smetrics |
|---|---|---|---|---|
| Short (7 chars) | **37** | 67 | 123 | 46 |
| Medium (25 chars) | **136** | 243 | 521 | 201 |
| Long (117 chars) | **1,530** | 2,763 | 4,429 | 2,594 |
| Unicode | **75** | 142 | 302 | 134 |
| Identical | 98 | **33** | 444 | 137 |
| Single char | **17** | 19 | 28 | **17** |

### Performance: Hamming Distance (ns/op)

| Scenario | strsim | go-edlib | strutil |
|---|---|---|---|
| Short | **12** | 14 | 14 |
| Medium | **37** | 42 | 41 |
| Identical | **29** | 32 | 32 |

### Performance: Batch Find Best Match (1000 candidates, Jaro-Winkler)

| Library | ns/op | B/op | allocs/op |
|---|---|---|---|
| strsim (FindBestMatch) | 121,490 | 43,520 | 2,000 |
| go-edlib (manual loop) | 188,210 | 280,322 | 2,000 |
| strutil (manual loop) | 338,976 | 168,322 | 6,880 |
| smetrics (manual loop) | 163,237 | 43,520 | 2,000 |
| **strsim ASCII (FindBestMatch)** | **106,343** | 43,520 | 2,000 |

### strsim ASCIIOnly Fast Path

strsim supports an `ASCIIOnly` mode that skips rune conversion for pure-ASCII input. This trades Unicode correctness for speed: use it only when you know your input is ASCII.

| Algorithm | Rune (ns/op) | ASCII (ns/op) | Speedup |
|---|---|---|---|
| Hamming | 113 | 10 | **11.3x** |
| Damerau-Levenshtein | 17,189 | 6,224 | **2.8x** |
| Jaro-Winkler | 325 | 331 | 1.0x |
| Levenshtein | 2,666 | 2,645 | 1.0x |

Input: 43-char ASCII strings. Since v0.2.0 the rune path of Jaro and Jaro-Winkler is as fast as the byte path, so `ASCIIOnly` pays off for Hamming and Damerau-Levenshtein. In batch matching it still saves about 12% (106k against 121k ns).

### Performance: Soundex (ns/op)

| Input | strsim | smetrics |
|---|---|---|
| "Robert" | 25 | **15** |
| "Washington" | 35 | **17** |
| "Schwarzenegger" | 45 | **16** |

smetrics Soundex is faster but has bugs: **panics on empty input** and produces **incorrect results for Ashcraft** (A226 instead of A261: it doesn't treat H and W as transparent, as the American Soundex specification requires).

## Key Findings

1. **smetrics operates on bytes, not runes**, so it produces incorrect Levenshtein distance for non-ASCII input (9 instead of 3 for Japanese strings)
2. **go-edlib and strutil return NaN** for `Similarity("", "")`, a division by zero in normalization
3. **smetrics Soundex panics** on empty input and gives wrong results for "Ashcraft" (H/W transparency bug)
4. **strsim Jaro-Winkler is the fastest**, or tied, in every scenario except identical strings, where go-edlib returns early
5. **strsim Hamming is fastest** across all scenarios
6. **strsim is the only library** combining edit distance + token-based + phonetic (15 algorithms)
7. **strsim batch matching is the fastest**: `FindBestMatch` takes 121k ns/op, 1.3x faster than a loop over smetrics (163k), and 106k with `ASCIIOnly`
8. **smetrics wins Levenshtein** on ASCII strings due to a tighter inner loop, but with broken Unicode, and **wins Soundex**, with the bugs above

## Methodology

- All benchmarks use Go's `testing.Benchmark` for accurate results
- Same inputs across all libraries per scenario
- Deterministic inputs (no randomness)
- Unicode correctness verified by comparing rune-level vs byte-level distance
- Empty string handling tested explicitly
- Run on the same machine; each number is the median of three runs of `go run .`

## Reproducing

```bash
git clone https://github.com/jcoruiz/strsim-benchmark.git
cd strsim-benchmark
go run .
```
