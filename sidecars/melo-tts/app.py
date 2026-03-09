import json
import os
import tempfile
import threading
from pathlib import Path
from typing import Dict, List, Optional

from fastapi import FastAPI, HTTPException, Response
from pydantic import BaseModel

from melo.api import TTS


def env_float(name: str, default: float) -> float:
    raw = os.getenv(name, "").strip()
    if not raw:
        return default
    try:
        return float(raw)
    except ValueError:
        return default


def env_int(name: str, default: int) -> int:
    raw = os.getenv(name, "").strip()
    if not raw:
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def normalized_text(raw: str) -> str:
    text = " ".join(str(raw or "").strip().split())
    if not text:
        return ""
    max_runes = env_int("MELO_MAX_TEXT_RUNES", 80)
    return "".join(list(text)[:max_runes])


def normalize_language(raw: str) -> str:
    key = str(raw or "").strip().lower()
    aliases = {
        "melo-zh": "ZH",
        "zh": "ZH",
        "zh-cn": "ZH",
        "melo-en": "EN",
        "en": "EN",
        "en-us": "EN-US",
        "en_default": "EN-Default",
        "es": "ES",
        "fr": "FR",
        "jp": "JP",
        "ja": "JP",
        "kr": "KR",
        "ko": "KR",
    }
    if key in aliases:
        return aliases[key]
    if raw:
        return str(raw).strip().upper()
    return os.getenv("MELO_DEFAULT_LANGUAGE", "ZH").strip().upper() or "ZH"


def normalize_format(response_format: Optional[str], format_name: Optional[str]) -> str:
    raw = str(response_format or format_name or "wav").strip().lower()
    if raw == "wav":
        return "wav"
    raise HTTPException(status_code=400, detail="melo sidecar currently supports wav only")


class SpeechRequest(BaseModel):
    model: Optional[str] = None
    voice: Optional[str] = None
    input: str
    format: Optional[str] = None
    response_format: Optional[str] = None


class VoiceOption(BaseModel):
    id: str
    label: str
    language: str = "ZH"
    speaker: str = "ZH"
    speed: float = 1.0
    description: str = ""
    isDefault: bool = False


class ModelRegistry:
    def __init__(self) -> None:
        self._models: Dict[str, TTS] = {}
        self._locks: Dict[str, threading.Lock] = {}
        self._guard = threading.Lock()
        self.device = os.getenv("MELO_DEVICE", "cpu").strip() or "cpu"

    def get(self, language: str) -> TTS:
        with self._guard:
            model = self._models.get(language)
            if model is None:
                model = TTS(language=language, device=self.device)
                self._models[language] = model
            if language not in self._locks:
                self._locks[language] = threading.Lock()
            return model

    def lock_for(self, language: str) -> threading.Lock:
        self.get(language)
        return self._locks[language]

    def loaded_languages(self) -> List[str]:
        with self._guard:
            return sorted(self._models.keys())


def default_voice_options() -> List[VoiceOption]:
    language = normalize_language(os.getenv("MELO_DEFAULT_LANGUAGE", "ZH"))
    speaker = os.getenv(f"MELO_SPEAKER_{language.upper()}", "").strip() or language
    return [
        VoiceOption(id="warm", label="温柔", language=language, speaker=speaker, speed=0.96, isDefault=True, description="默认柔和语速"),
        VoiceOption(id="bright", label="清亮", language=language, speaker=speaker, speed=1.06, description="更快一点的语速"),
        VoiceOption(id="steady", label="沉稳", language=language, speaker=speaker, speed=1.00, description="标准语速"),
    ]


def load_voice_options() -> List[VoiceOption]:
    raw = os.getenv("MELO_VOICES_JSON", "").strip()
    if not raw:
        return default_voice_options()

    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"invalid MELO_VOICES_JSON: {exc}") from exc

    if not isinstance(payload, list):
        raise RuntimeError("MELO_VOICES_JSON must be a JSON array")

    items: List[VoiceOption] = []
    seen = set()
    for item in payload:
        if not isinstance(item, dict):
            continue
        voice_id = " ".join(str(item.get("id", "")).strip().split())
        if not voice_id or voice_id in seen:
            continue
        seen.add(voice_id)
        label = " ".join(str(item.get("label") or voice_id).strip().split()) or voice_id
        language = normalize_language(str(item.get("language") or os.getenv("MELO_DEFAULT_LANGUAGE", "ZH")))
        speaker = str(item.get("speaker") or os.getenv(f"MELO_SPEAKER_{language.upper()}", "") or language).strip() or language
        speed = float(item.get("speed") or 1.0)
        description = " ".join(str(item.get("description", "")).strip().split())
        is_default = bool(item.get("isDefault"))
        items.append(
            VoiceOption(
                id=voice_id,
                label=label,
                language=language,
                speaker=speaker,
                speed=speed,
                description=description,
                isDefault=is_default,
            )
        )

    if not items:
        return default_voice_options()

    if not any(item.isDefault for item in items):
        items[0].isDefault = True
    return items


registry = ModelRegistry()
VOICE_OPTIONS = load_voice_options()
VOICE_OPTION_MAP = {item.id: item for item in VOICE_OPTIONS}

app = FastAPI(title="ScoreHub MeloTTS Sidecar", version="0.2.0")


@app.on_event("startup")
def preload_languages() -> None:
    raw = os.getenv("MELO_PRELOAD_LANGUAGES", "").strip()
    if raw:
        languages = [normalize_language(item) for item in raw.split(",") if item.strip()]
    else:
        languages = sorted({item.language for item in VOICE_OPTIONS})
    for language in languages:
        try:
            registry.get(language)
        except Exception:
            continue


def default_voice_option() -> VoiceOption:
    for item in VOICE_OPTIONS:
        if item.isDefault:
            return item
    return VOICE_OPTIONS[0]


def resolve_voice_option(voice_id: Optional[str], model_name: Optional[str]) -> VoiceOption:
    normalized_voice = " ".join(str(voice_id or "").strip().split())
    if normalized_voice:
        item = VOICE_OPTION_MAP.get(normalized_voice)
        if item:
            return item
        raise HTTPException(status_code=400, detail="invalid voice")

    if model_name:
        wanted_language = normalize_language(model_name)
        for item in VOICE_OPTIONS:
            if item.language == wanted_language and item.isDefault:
                return item
        for item in VOICE_OPTIONS:
            if item.language == wanted_language:
                return item

    return default_voice_option()


def resolve_speaker_id(model: TTS, requested_speaker: str) -> tuple[str, int]:
    speaker_ids = getattr(model.hps.data, "spk2id", {}) or {}
    if not speaker_ids:
        raise HTTPException(status_code=500, detail="melo speaker ids unavailable")

    if requested_speaker in speaker_ids:
        return requested_speaker, speaker_ids[requested_speaker]

    upper = requested_speaker.upper()
    if upper in speaker_ids:
        return upper, speaker_ids[upper]

    lower_map = {str(name).lower(): name for name in speaker_ids.keys()}
    lowered = requested_speaker.lower()
    if lowered in lower_map:
        name = lower_map[lowered]
        return str(name), speaker_ids[name]

    first_name = next(iter(speaker_ids.keys()))
    return str(first_name), speaker_ids[first_name]


@app.get("/health")
@app.get("/v1/health")
def health() -> dict:
    return {
        "ok": True,
        "device": registry.device,
        "loadedLanguages": registry.loaded_languages(),
        "voices": len(VOICE_OPTIONS),
    }


@app.get("/audio/voices")
@app.get("/v1/audio/voices")
def list_voices() -> dict:
    default_item = default_voice_option()
    return {
        "defaultVoice": default_item.id,
        "voices": [
            {
                "id": item.id,
                "label": item.label,
                "language": item.language,
                "description": item.description,
                "isDefault": item.isDefault,
            }
            for item in VOICE_OPTIONS
        ],
    }


@app.post("/audio/speech")
@app.post("/v1/audio/speech")
def create_speech(payload: SpeechRequest) -> Response:
    text = normalized_text(payload.input)
    if not text:
        raise HTTPException(status_code=400, detail="input is required")

    response_format = normalize_format(payload.response_format, payload.format)
    selected = resolve_voice_option(payload.voice, payload.model)

    try:
        model = registry.get(selected.language)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"load melo model failed: {exc}") from exc

    speaker_name, speaker_id = resolve_speaker_id(model, selected.speaker)

    fd, output_path = tempfile.mkstemp(suffix=f".{response_format}")
    os.close(fd)
    try:
        with registry.lock_for(selected.language):
            model.tts_to_file(text, speaker_id, output_path, speed=selected.speed)
        audio = Path(output_path).read_bytes()
    except HTTPException:
        raise
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"melo synthesis failed: {exc}") from exc
    finally:
        try:
            os.remove(output_path)
        except OSError:
            pass

    headers = {
        "X-TTS-Provider": "melo-tts",
        "X-TTS-Language": selected.language,
        "X-TTS-Speaker": speaker_name,
        "X-TTS-Voice": selected.id,
        "X-TTS-Speed": f"{selected.speed:.2f}",
    }
    return Response(content=audio, media_type="audio/wav", headers=headers)
