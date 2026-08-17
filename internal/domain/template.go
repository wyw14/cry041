package domain

import (
	"errors"
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
		ApplicableRisk: t.ApplicableRisk, ApplicableEnvironments: t.ApplicableEnvironments,
		Items: t.Items}
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
	_ = target
	return previous
}
