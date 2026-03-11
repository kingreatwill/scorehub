import json
import os
import shutil
import subprocess
import tempfile
import threading
import urllib.error
import urllib.request
from pathlib import Path
from typing import List, Optional, Tuple

from fastapi import FastAPI, HTTPException, Response
from pydantic import BaseModel


_DOWNLOAD_LOCKS: dict[str, threading.Lock] = {}
_DOWNLOAD_GUARD = threading.Lock()


def env_bool(name: str, default: bool) -> bool:
    raw = os.getenv(name, "").strip().lower()
    if not raw:
        return default
    if raw in {"1", "true", "yes", "y", "on"}:
        return True
    if raw in {"0", "false", "no", "n", "off"}:
        return False
    return default


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


def env_optional_int(name: str) -> Optional[int]:
    raw = os.getenv(name, "").strip()
    if not raw:
        return None
    try:
        return int(raw)
    except ValueError:
        return None


def normalize_key(raw: str) -> str:
    return " ".join(str(raw or "").strip().split()).lower()


def normalized_text(raw: str) -> str:
    text = " ".join(str(raw or "").strip().split())
    if not text:
        return ""
    max_runes = env_int("PIPER_MAX_TEXT_RUNES", 120)
    return "".join(list(text)[:max_runes])


def normalize_format(response_format: Optional[str], format_name: Optional[str]) -> str:
    raw = str(response_format or format_name or "wav").strip().lower()
    if raw == "wav":
        return "wav"
    raise HTTPException(status_code=400, detail="piper sidecar currently supports wav only")


def models_dir() -> Path:
    raw = os.getenv("PIPER_MODELS_DIR", "/models").strip() or "/models"
    return Path(raw)


def resolve_model_path(raw: str) -> str:
    text = " ".join(str(raw or "").strip().split())
    if not text:
        return ""
    path = Path(text)
    if path.is_absolute():
        return str(path)
    return str(models_dir() / path)


def parse_optional_int(raw: object) -> Optional[int]:
    if raw is None:
        return None
    text = str(raw).strip()
    if not text:
        return None
    try:
        return int(text)
    except ValueError:
        return None


def normalize_url(raw: object) -> str:
    return " ".join(str(raw or "").strip().split())


class SpeechRequest(BaseModel):
    model: Optional[str] = None
    voice: Optional[str] = None
    input: str
    format: Optional[str] = None
    response_format: Optional[str] = None


class VoiceOption(BaseModel):
    id: str
    label: str
    language: str = "zh-CN"
    modelPath: str
    modelAlias: str = ""
    speaker: Optional[int] = None
    lengthScale: float = 1.0
    noiseScale: float = 0.667
    noiseW: float = 0.8
    sampleRate: int = 22050
    description: str = ""
    isDefault: bool = False
    downloadURL: str = ""
    configURL: str = ""


def default_voice_options() -> List[VoiceOption]:
    voice_id = " ".join(os.getenv("PIPER_DEFAULT_VOICE_ID", "default").strip().split()) or "default"
    label = " ".join(os.getenv("PIPER_DEFAULT_VOICE_LABEL", "默认").strip().split()) or voice_id
    language = " ".join(os.getenv("PIPER_DEFAULT_LANGUAGE", "zh-CN").strip().split()) or "zh-CN"
    model_alias = " ".join(os.getenv("PIPER_DEFAULT_MODEL", voice_id).strip().split()) or voice_id
    model_path = resolve_model_path(os.getenv("PIPER_MODEL_PATH", "model.onnx"))
    description = " ".join(os.getenv("PIPER_DEFAULT_VOICE_DESCRIPTION", "").strip().split())
    download_url = normalize_url(os.getenv("PIPER_MODEL_DOWNLOAD_URL", ""))
    config_url = normalize_url(os.getenv("PIPER_MODEL_CONFIG_URL", ""))
    return [
        VoiceOption(
            id=voice_id,
            label=label,
            language=language,
            modelPath=model_path,
            modelAlias=model_alias,
            speaker=env_optional_int("PIPER_DEFAULT_SPEAKER"),
            lengthScale=env_float("PIPER_LENGTH_SCALE", 1.0),
            noiseScale=env_float("PIPER_NOISE_SCALE", 0.667),
            noiseW=env_float("PIPER_NOISE_W", 0.8),
            sampleRate=env_int("PIPER_SAMPLE_RATE", 22050),
            description=description,
            isDefault=True,
            downloadURL=download_url,
            configURL=config_url,
        )
    ]


def load_voice_options() -> List[VoiceOption]:
    raw = os.getenv("PIPER_VOICES_JSON", "").strip()
    if not raw:
        return default_voice_options()

    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"invalid PIPER_VOICES_JSON: {exc}") from exc

    if not isinstance(payload, list):
        raise RuntimeError("PIPER_VOICES_JSON must be a JSON array")

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
        language = " ".join(str(item.get("language") or os.getenv("PIPER_DEFAULT_LANGUAGE", "zh-CN")).strip().split()) or "zh-CN"
        model_path = resolve_model_path(str(item.get("modelPath") or item.get("path") or item.get("voicePath") or ""))
        if not model_path:
            continue
        model_alias = " ".join(str(item.get("modelAlias") or item.get("model") or item.get("providerModel") or voice_id).strip().split()) or voice_id
        description = " ".join(str(item.get("description", "")).strip().split())
        items.append(
            VoiceOption(
                id=voice_id,
                label=label,
                language=language,
                modelPath=model_path,
                modelAlias=model_alias,
                speaker=parse_optional_int(item.get("speaker")),
                lengthScale=float(item.get("lengthScale") or item.get("speed") or 1.0),
                noiseScale=float(item.get("noiseScale") or 0.667),
                noiseW=float(item.get("noiseW") or 0.8),
                sampleRate=int(item.get("sampleRate") or 22050),
                description=description,
                isDefault=bool(item.get("isDefault")),
                downloadURL=normalize_url(item.get("downloadURL") or item.get("downloadUrl") or item.get("url")),
                configURL=normalize_url(item.get("configURL") or item.get("configUrl") or item.get("jsonURL") or item.get("jsonUrl")),
            )
        )

    if not items:
        return default_voice_options()

    if not any(item.isDefault for item in items):
        items[0].isDefault = True
    return items


VOICE_OPTIONS = load_voice_options()
app = FastAPI(title="ScoreHub Piper Sidecar", version="0.3.0")


def default_voice_option() -> VoiceOption:
    for item in VOICE_OPTIONS:
        if item.isDefault:
            return item
    return VOICE_OPTIONS[0]


def resolve_voice_option(voice_id: Optional[str], model_name: Optional[str]) -> VoiceOption:
    normalized_voice = normalize_key(voice_id or "")
    if normalized_voice:
        for item in VOICE_OPTIONS:
            if normalize_key(item.id) == normalized_voice:
                return item
        raise HTTPException(status_code=400, detail="invalid voice")

    normalized_model = normalize_key(model_name or "")
    if normalized_model:
        for item in VOICE_OPTIONS:
            if normalize_key(item.modelAlias) == normalized_model:
                return item
        for item in VOICE_OPTIONS:
            if normalize_key(item.id) == normalized_model:
                return item
        for item in VOICE_OPTIONS:
            if normalize_key(item.language) == normalized_model and item.isDefault:
                return item
        for item in VOICE_OPTIONS:
            if normalize_key(item.language) == normalized_model:
                return item

    return default_voice_option()


def model_config_path(voice: VoiceOption) -> Path:
    return Path(f"{voice.modelPath}.json")


def voice_is_ready(voice: VoiceOption) -> bool:
    return Path(voice.modelPath).exists() and model_config_path(voice).exists()


def missing_voice_models() -> List[str]:
    return [item.id for item in VOICE_OPTIONS if not voice_is_ready(item)]


def download_lock_for(model_path: str) -> threading.Lock:
    with _DOWNLOAD_GUARD:
        lock = _DOWNLOAD_LOCKS.get(model_path)
        if lock is None:
            lock = threading.Lock()
            _DOWNLOAD_LOCKS[model_path] = lock
        return lock


def inferred_download_urls(voice: VoiceOption) -> Tuple[str, str]:
    basename = Path(voice.modelPath).name
    if not basename.endswith(".onnx"):
        return "", ""

    stem = basename[:-5]
    parts = stem.split("-")
    if len(parts) < 3:
        return "", ""

    locale = parts[0]
    quality = parts[-1]
    voice_name = "-".join(parts[1:-1])
    if not locale or not quality or not voice_name:
        return "", ""

    root = locale.split("_", 1)[0].lower()
    base = f"https://huggingface.co/rhasspy/piper-voices/resolve/main/{root}/{locale}/{voice_name}/{quality}/{basename}"
    return base, f"{base}.json"


def resolve_download_urls(voice: VoiceOption) -> Tuple[str, str]:
    model_url = voice.downloadURL
    config_url = voice.configURL
    inferred_model_url, inferred_config_url = inferred_download_urls(voice)
    if not model_url:
        model_url = inferred_model_url
    if not config_url:
        config_url = inferred_config_url if not model_url.endswith(".onnx") else f"{model_url}.json"
    return model_url, config_url


def download_file(url: str, destination: Path) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    timeout_secs = env_int("PIPER_DOWNLOAD_TIMEOUT_SECS", 120)
    with urllib.request.urlopen(url, timeout=timeout_secs) as response:
        with tempfile.NamedTemporaryFile(delete=False, dir=str(destination.parent), suffix=".part") as handle:
            shutil.copyfileobj(response, handle)
            temp_name = handle.name
    Path(temp_name).replace(destination)


def auto_download_enabled() -> bool:
    return env_bool("PIPER_AUTO_DOWNLOAD", True)


def preload_models_enabled() -> bool:
    return env_bool("PIPER_PRELOAD_MODELS", False)


def ensure_voice_model_available(voice: VoiceOption) -> None:
    if voice_is_ready(voice):
        return
    if not auto_download_enabled():
        raise HTTPException(status_code=500, detail=f"piper model not found for voice: {voice.id}")

    model_url, config_url = resolve_download_urls(voice)
    if not model_url or not config_url:
        raise HTTPException(status_code=500, detail=f"missing download url for voice: {voice.id}")

    lock = download_lock_for(voice.modelPath)
    with lock:
        if voice_is_ready(voice):
            return

        model_path = Path(voice.modelPath)
        config_path = model_config_path(voice)
        try:
            if not model_path.exists():
                download_file(model_url, model_path)
            if not config_path.exists():
                download_file(config_url, config_path)
        except urllib.error.URLError as exc:
            raise HTTPException(status_code=502, detail=f"download piper model failed: {exc}") from exc
        except OSError as exc:
            raise HTTPException(status_code=500, detail=f"write piper model failed: {exc}") from exc

        if not voice_is_ready(voice):
            raise HTTPException(status_code=502, detail=f"downloaded piper model incomplete for voice: {voice.id}")


def build_command(voice: VoiceOption, output_path: str) -> List[str]:
    cmd = [
        os.getenv("PIPER_BIN", "piper").strip() or "piper",
        "--model",
        voice.modelPath,
        "--output_file",
        output_path,
        "--length_scale",
        f"{voice.lengthScale}",
        "--noise_scale",
        f"{voice.noiseScale}",
        "--noise_w",
        f"{voice.noiseW}",
    ]
    if voice.speaker is not None:
        cmd.extend(["--speaker", str(voice.speaker)])
    return cmd


@app.on_event("startup")
def preload_models() -> None:
    if not preload_models_enabled():
        return
    for item in VOICE_OPTIONS:
        ensure_voice_model_available(item)


@app.get("/health")
@app.get("/v1/health")
def health() -> dict:
    default_item = default_voice_option()
    missing_models = missing_voice_models()
    return {
        "ok": True,
        "ready": len(missing_models) == 0,
        "autoDownload": auto_download_enabled(),
        "preloadModels": preload_models_enabled(),
        "defaultVoice": default_item.id,
        "voices": len(VOICE_OPTIONS),
        "modelsDir": str(models_dir()),
        "missingModels": missing_models,
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
    ensure_voice_model_available(selected)

    fd, output_path = tempfile.mkstemp(suffix=f".{response_format}")
    os.close(fd)
    try:
        timeout_secs = env_int("PIPER_TIMEOUT_SECS", 30)
        result = subprocess.run(
            build_command(selected, output_path),
            input=text.encode("utf-8"),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=timeout_secs,
        )
        if result.returncode != 0:
            detail = result.stderr.decode("utf-8", errors="ignore").strip() or "piper command failed"
            raise HTTPException(status_code=502, detail=detail)
        audio = Path(output_path).read_bytes()
    except subprocess.TimeoutExpired as exc:
        raise HTTPException(status_code=504, detail="piper synthesis timed out") from exc
    finally:
        try:
            os.remove(output_path)
        except OSError:
            pass

    headers = {
        "X-TTS-Provider": "piper",
        "X-TTS-Language": selected.language,
        "X-TTS-Voice": selected.id,
        "X-TTS-Model": selected.modelAlias,
        "X-TTS-Length-Scale": f"{selected.lengthScale:.3f}",
    }
    if selected.speaker is not None:
        headers["X-TTS-Speaker"] = str(selected.speaker)
    return Response(content=audio, media_type="audio/wav", headers=headers)
