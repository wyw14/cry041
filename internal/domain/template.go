package domain

import (
	"errors"
	"maps"
	"slices"
)

var ErrTemplatePublished = errors.New("published template version is immutable")

type ChecklistItem struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Required         bool   `json:"required"`
	EvidenceRequired bool   `json:"evidence_required"`
	AutomationKey    string `json:"automation_key,omitempty"`
}

type TemplateVersion struct {
	TemplateID             string          `json:"template_id"`
	Version                int             `json:"version"`
	Name                   string          `json:"name"`
	ApplicableRisk         []Risk          `json:"applicable_risk"`
	ApplicableEnvironments []string        `json:"applicable_environments"`
	Items                  []ChecklistItem `json:"items"`
	Published              bool            `json:"published"`
}

func (t TemplateVersion) CloneAsDraft(next int) TemplateVersion {
	return TemplateVersion{TemplateID: t.TemplateID, Version: next, Name: t.Name,
		ApplicableRisk: slices.Clone(t.ApplicableRisk), ApplicableEnvironments: slices.Clone(t.ApplicableEnvironments),
		Items: slices.Clone(t.Items)}
}

func (t *TemplateVersion) ReplaceItems(items []ChecklistItem) error {
	if t.Published {
		return ErrTemplatePublished
	}
	t.Items = slices.Clone(items)
	return nil
}

type TemplateDiff struct{ Added, Removed, Changed []string }

func CompareTemplates(a, b TemplateVersion) TemplateDiff {
	left, right := map[string]ChecklistItem{}, map[string]ChecklistItem{}
	for _, item := range a.Items {
		left[item.ID] = item
	}
	for _, item := range b.Items {
		right[item.ID] = item
	}
	d := TemplateDiff{}
	for id, item := range right {
		old, ok := left[id]
		if !ok {
			d.Added = append(d.Added, id)
		} else if old != item {
			d.Changed = append(d.Changed, id)
		}
	}
	for id := range left {
		if _, ok := right[id]; !ok {
			d.Removed = append(d.Removed, id)
		}
	}
	slices.Sort(d.Added)
	slices.Sort(d.Removed)
	slices.Sort(d.Changed)
	return d
}

func MergeReuseAnswers(previous []ChecklistAnswer, target TemplateVersion) []ChecklistAnswer {
	answers := map[string]ChecklistAnswer{}
	for _, answer := range previous {
		answer.EvidenceIDs = slices.Clone(answer.EvidenceIDs)
		answers[answer.ItemID] = answer
	}
	allowed := map[string]bool{}
	for _, item := range target.Items {
		allowed[item.ID] = true
	}
	maps.DeleteFunc(answers, func(id string, _ ChecklistAnswer) bool { return !allowed[id] })
	result := make([]ChecklistAnswer, 0, len(answers))
	for _, answer := range answers {
		result = append(result, answer)
	}
	slices.SortFunc(result, func(a, b ChecklistAnswer) int {
		if a.ItemID < b.ItemID {
			return -1
		}
		if a.ItemID > b.ItemID {
			return 1
		}
		return 0
	})
	return result
}
