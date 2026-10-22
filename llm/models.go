package llm

import (
	"cmp"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ModelInfo is the advertised model ID and provider ownership from discovery.
type ModelInfo struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
}

// ModelSelection identifies the advertised route chosen for a client.
type ModelSelection struct {
	Provider string
	ID       string
}

// Version components are bounded to six digits, with no noncanonical leading
// zeroes. Anchoring excludes unverified aliases and non-text model variants.
const modelNumber = `(0|[1-9][0-9]{0,5})`

var (
	opusModel   = regexp.MustCompile(`^claude-opus-(` + modelNumber + `(?:-` + modelNumber + `){0,2})(?:-([0-9]{8}))?$`)
	gptModel    = regexp.MustCompile(`^gpt-(` + modelNumber + `(?:\.` + modelNumber + `){0,2})(-codex)?$`)
	geminiModel = regexp.MustCompile(`^gemini-(` + modelNumber + `(?:\.` + modelNumber + `){0,2})-(flash|pro)(?:-(high|low|lite))?$`)
)

type modelRank struct {
	provider, preference int
	version              [3]int
	variant              int
}

// SelectModel selects a supported advertised model using provider-first policy.
// It neither changes the catalog nor manufactures IDs. Owner metadata is
// authoritative even when a model's prefix resembles another provider.
func SelectModel(models []ModelInfo) (ModelSelection, error) {
	var selected ModelSelection
	var best modelRank
	for _, model := range models {
		rank, ok := rankModel(model)
		if !ok {
			continue
		}
		comparison := compareModelRank(rank, best)
		if selected.ID == "" || comparison > 0 || comparison == 0 && model.ID < selected.ID {
			best = rank
			selected = ModelSelection{Provider: model.OwnedBy, ID: model.ID}
		}
	}
	if selected.ID == "" {
		return ModelSelection{}, fmt.Errorf("%w: no supported model advertised; set DEFINE_LLM_MODEL to use an explicit model", ErrUnavailable)
	}
	return selected, nil
}

func rankModel(model ModelInfo) (modelRank, bool) {
	var rank modelRank
	// Discovery enforces this too; the pure selector remains bounded for callers
	// supplying a catalog directly.
	if len(model.ID) > 256 {
		return rank, false
	}
	switch model.OwnedBy {
	case "anthropic":
		match := opusModel.FindStringSubmatch(model.ID)
		if match == nil {
			return rank, false
		}
		date := match[len(match)-1]
		if date != "" {
			if _, err := time.Parse("20060102", date); err != nil {
				return rank, false
			}
		}
		rank.provider = 3
		rank.version = modelVersion(match[1], "-")
	case "openai":
		match := gptModel.FindStringSubmatch(model.ID)
		if match == nil {
			return rank, false
		}
		rank.provider = 2
		rank.version = modelVersion(match[1], ".")
		switch rank.version {
		case [3]int{5, 6, 0}:
			rank.preference = 2
		case [3]int{6, 0, 0}:
			rank.preference = 1
		}
		if match[len(match)-1] == "" {
			rank.variant = 1
		}
	case "antigravity":
		match := geminiModel.FindStringSubmatch(model.ID)
		if match == nil {
			return rank, false
		}
		rank.provider = 1
		rank.version = modelVersion(match[1], ".")
		if match[len(match)-2] == "flash" {
			rank.preference = 1
		}
		switch match[len(match)-1] {
		case "high":
			rank.variant = 3
		case "":
			rank.variant = 2
		case "low":
			rank.variant = 1
		}
	default:
		return rank, false
	}
	return rank, true
}

func modelVersion(raw, separator string) [3]int {
	var version [3]int
	for i, part := range strings.Split(raw, separator) {
		// The anchored grammar already guarantees bounded decimal integers.
		version[i], _ = strconv.Atoi(part)
	}
	return version
}

func compareModelRank(a, b modelRank) int {
	for _, pair := range [][2]int{
		{a.provider, b.provider}, {a.preference, b.preference},
		{a.version[0], b.version[0]}, {a.version[1], b.version[1]}, {a.version[2], b.version[2]},
		{a.variant, b.variant},
	} {
		if difference := cmp.Compare(pair[0], pair[1]); difference != 0 {
			return difference
		}
	}
	return 0
}
