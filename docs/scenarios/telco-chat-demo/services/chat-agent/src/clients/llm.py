from langchain.chat_models import init_chat_model
from langchain_core.language_models import BaseChatModel

from src.config import settings


def get_model() -> BaseChatModel:
    return init_chat_model(
        model=settings.openai_model,
        model_provider="openai",
        api_key=settings.openai_api_key,
        streaming=True,
    )
