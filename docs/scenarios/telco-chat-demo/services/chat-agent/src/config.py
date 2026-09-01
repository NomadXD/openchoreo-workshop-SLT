from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        case_sensitive=False,
        extra="allow",
    )

    port: int = 8080

    # OpenAI. Required — checked (fail-fast) at startup in main.py's lifespan
    # rather than enforced here, so a clear RuntimeError is raised instead of
    # a raw pydantic ValidationError.
    openai_api_key: str = ""
    openai_model: str = "gpt-4o-mini"

    # Backend services this agent calls tools against.
    subscription_service_url: str = ""
    network_ops_service_url: str = ""

    log_level: str = "INFO"


settings = Settings()
