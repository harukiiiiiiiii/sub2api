package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDefaultImageModelSettingsLiveRead(t *testing.T) {
	ctx := context.Background()
	repo := &panelRateLimitSettingRepo{}
	svc := NewSettingService(repo, &config.Config{})
	require.Equal(t, openAIImagesDefaultToolModel, svc.GetOpenAIImagesDefaultModel(ctx))
	require.NoError(t, svc.UpdateSettings(ctx, &SystemSettings{OpenAIImagesDefaultModel: "gpt-image-2"}))
	require.Equal(t, "gpt-image-2", svc.GetOpenAIImagesDefaultModel(ctx))
	require.NoError(t, svc.UpdateSettingsOmitting(ctx, &SystemSettings{}, OmittedSettingKeys{SettingKeyOpenAIImagesDefaultModel: {}}))
	require.Equal(t, "gpt-image-2", svc.GetOpenAIImagesDefaultModel(ctx))
	require.Error(t, svc.UpdateSettings(ctx, &SystemSettings{OpenAIImagesDefaultModel: "gpt-5.6-luna"}))
	require.Equal(t, "gpt-image-2", svc.GetOpenAIImagesDefaultModel(ctx))
	require.NoError(t, svc.UpdateSettings(ctx, &SystemSettings{OpenAIImagesDefaultModel: "gpt-image-2.5-sunburst"}))
	require.Equal(t, "gpt-image-2.5-sunburst", svc.GetOpenAIImagesDefaultModel(ctx))
	require.NoError(t, repo.Set(ctx, SettingKeyOpenAIImagesDefaultModel, "invalid"))
	require.Equal(t, openAIImagesDefaultToolModel, svc.GetOpenAIImagesDefaultModel(ctx))
	repo.getValueErr = errors.New("unavailable")
	require.Equal(t, openAIImagesDefaultToolModel, svc.GetOpenAIImagesDefaultModel(ctx))
	require.Equal(t, openAIImagesDefaultToolModel, (*SettingService)(nil).GetOpenAIImagesDefaultModel(ctx))
}

func TestConfiguredImageModelRequests(t *testing.T) {
	repo := &panelRateLimitSettingRepo{values: map[string]string{SettingKeyOpenAIImagesDefaultModel: "gpt-image-2.5-sunburst"}}
	svc := &OpenAIGatewayService{settingService: &SettingService{settingRepo: repo}}
	for _, tc := range []struct{ body, want string }{
		{`{"prompt":"cat"}`, "gpt-image-2.5-sunburst"},
		{`{"prompt":"cat","model":"gpt-image-2"}`, "gpt-image-2"},
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
		c.Request.Header.Set("Content-Type", "application/json")
		parsed, err := svc.ParseOpenAIImagesRequest(c, []byte(tc.body))
		require.NoError(t, err)
		require.Equal(t, tc.want, parsed.Model)
	}
	for _, tc := range []struct{ body, want string }{
		{`{"model":"gpt-5.6-luna","tools":[{"type":"image_generation"}]}`, "gpt-image-2.5-sunburst"},
		{`{"model":"gpt-5.6-luna","tools":[{"type":"image_generation","model":"gpt-image-2"}]}`, "gpt-image-2"},
		{`{"model":"gpt-image-2","tools":[{"type":"image_generation"}]}`, "gpt-image-2"},
	} {
		body := []byte(tc.body)
		result, _, err := normalizeOpenAIResponsesWebSocketCompatibilityBody(body, &Account{Platform: "openai", Type: AccountTypeAPIKey}, false, svc.defaultImageModelForBody(context.Background(), body))
		require.NoError(t, err)
		require.Equal(t, tc.want, gjson.GetBytes(result, "tools.0.model").String())
	}
	body := map[string]any{"model": "gpt-5.6-luna"}
	require.True(t, ensureOpenAIResponsesImageGenerationTool(body, "gpt-image-2.5-sunburst"))
	require.Equal(t, "gpt-image-2.5-sunburst", body["tools"].([]any)[0].(map[string]any)["model"])
	require.Equal(t, "gpt-5.6-luna", body["model"])
}
