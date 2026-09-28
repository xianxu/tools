---
id: '000035'
status: done
started: 2026-08-29T14:56:14-07:00
created: 2026-08-29
updated: 2026-08-29
estimate_hours: 1.39
actual_hours: 0.85
---

# /pron with no language: infer the origin from ORIGIN, error only when it cannot be determined

## Problem

Operator, after using `#29`:

> improvement number one is just use `/pron` instead of `/pron fr`, basically
> infer origin language whereever possible and only error out on `/pron` if you
> can't determine.

`#29` shipped a DECLARED language, and the entry prints `ORIGIN French` two lines
above the prompt. So the tool displays the answer and then asks you to retype it
as a code — and to know that Japanese is `ja`.
