package vertex

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ProjectID                    string `mapstructure:"project_id"`
	Location                     string `mapstructure:"location"`
	DatasetID                    string `mapstructure:"dataset_id"`
	IndexID                      string `mapstructure:"index_id"`
	EndpointID                   string `mapstructure:"endpoint_id"`
	DeployedIndexID              string `mapstructure:"deployed_index_id"`
	GenAIUseVertexAI             bool   `mapstructure:"google_genai_use_vertexai"`
	GoogleApplicationCredentials string `mapstructure:"google_application_credentials"`
	EndpointPublicDomainName     string `mapstructure:"endpoint_public_domain_name"`
	Limit                        int    `mapstructure:"limit"`
	Query                        string `mapstructure:"query"`
	Prompt                       string `mapstructure:"prompt"`
	SecurityPrompt               string `mapstructure:"security_prompt"`
	BigReviewEmbeddings          string `mapstructure:"big_review_embeddings"`
	BigHotels                    string `mapstructure:"big_hotels"`
	BigReviews                   string `mapstructure:"big_reviews"`
	PreferredModel               string `mapstructure:"preferred_model"`
	GeminiModel                  string `mapstructure:"gemini_model"`
	GrokAPIKey                   string `mapstructure:"grok_api_key"`
	GrokModel                    string `mapstructure:"grok_model"`
	OpenAIAPIKey                 string `mapstructure:"openai_api_key"`
	OpenAIModel                  string `mapstructure:"openai_model"`
	GooglePlacesAPIKey           string `mapstructure:"google_places_api_key"`
	CORSAllowedOrigins           string `mapstructure:"cors_allowed_origins"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	v.AutomaticEnv()

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
