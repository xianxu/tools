---
id: '000027'
status: done
started: 2026-08-28T00:38:19-07:00
created: 2026-08-28
updated: 2026-08-28
estimate_hours: 0.93
actual_hours: 1.89
---

# pronunciation locale and language as parameters, not literals

## Problem

Split from **#18 M1**, which said it plainly: *"M1 is independently shippable and
does not wait on #10."* #18's remaining scope (deck language dimension,
inflection/lemma identity, gender, agreement-safe distractors) is blocked on
`#10`/`#12`; this half is blocked on nothing, and two other issues are waiting on
the concept it introduces.

`AudioCandidates` (`cmd/define/audiourl.go:41`) writes the language as a
**literal**:

```go
out = append(out, audioBase+"/pronunciation/2022-03-02/audio/"+shard+"/"+esc+"_en_"+locale+"_"+n+".mp3")
```

So `madrugar` is only ever requested as an English word. Measured on 2026-08-22 —
all four candidates 404 while the recording sits two characters away:

```
madrugar_en_us_1.mp3   404      ← what define asks for
madrugar--_us_1.mp3    404
madrugar_es_es_1.mp3   200      ← what exists
madrugar_es_us_1.mp3   200
```

Coverage is real, not incidental: `sobremesa`, `empalagoso`, `chapucero`,
`desvelarse` all return 200 on `es_es`.
