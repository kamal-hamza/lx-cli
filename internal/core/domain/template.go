package domain

import (
	"regexp"
	"strings"
)

type TemplateHeader struct {
	Title    string `yaml:"templateTitle"`
	Date     string `yaml:"date"`
	Slug     string `yaml:"-"`
	Filename string `yaml:"-"`
}

type TemplateBody struct {
	Header  TemplateHeader
	Content string
}

func GenerateTemplateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces and special characters with hyphens, but preserve underscores and hyphens
	reg := regexp.MustCompile(`[^a-z0-9_-]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens and underscores
	slug = strings.Trim(slug, "-_")

	// Collapse multiple hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	return slug
}
