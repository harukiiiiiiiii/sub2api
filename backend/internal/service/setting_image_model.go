package service

import (
	"context"
	"strings"
)

func validOpenAIImagesDefaultModel(model string) bool {
	switch model {
	case "gpt-image-1", "gpt-image-1.5", "gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst":
		return true
	}
	return false
}

func normalizeOpenAIImagesDefaultModel(model string) string {
	model = strings.TrimSpace(model)
	if validOpenAIImagesDefaultModel(model) {
		return model
	}
	return openAIImagesDefaultToolModel
}

// Read on demand so saved changes also apply to subsequent WebSocket turns.
func (s *SettingService) GetOpenAIImagesDefaultModel(ctx context.Context) string {
	if s == nil || s.settingRepo == nil {
		return openAIImagesDefaultToolModel
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIImagesDefaultModel)
	if err != nil {
		return openAIImagesDefaultToolModel
	}
	return normalizeOpenAIImagesDefaultModel(value)
}

func (s *OpenAIGatewayService) defaultImageModel(ctx context.Context) string {
	if s == nil {
		return openAIImagesDefaultToolModel
	}
	return s.settingService.GetOpenAIImagesDefaultModel(ctx)
}

func (s *OpenAIGatewayService) defaultImageModelForBody(ctx context.Context, body []byte) string {
	if !openAIRequestBodyImageGenerationToolNeedsNormalization(body) {
		return openAIImagesDefaultToolModel
	}
	return s.defaultImageModel(ctx)
}

func optionalDefaultImageModel(models []string) string {
	if len(models) > 0 {
		return normalizeOpenAIImagesDefaultModel(models[0])
	}
	return openAIImagesDefaultToolModel
}
