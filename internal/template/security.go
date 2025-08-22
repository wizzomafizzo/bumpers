package template

import "fmt"

const MaxTemplateSize = 10240 // 10KB

func ValidateTemplate(templateStr string) error {
	if len(templateStr) > MaxTemplateSize {
		return fmt.Errorf("template size %d exceeds maximum allowed size %d", len(templateStr), MaxTemplateSize)
	}
	return nil
}
