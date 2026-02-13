#!/usr/bin/env python3

import argparse
import json
from pathlib import Path


KEY_MAP = {
    "type": "GOOGLE_TYPE",
    "project_id": "GOOGLE_PROJECT_ID",
    "private_key_id": "GOOGLE_PRIVATE_KEY_ID",
    "private_key": "GOOGLE_PRIVATE_KEY",
    "client_email": "GOOGLE_CLIENT_EMAIL",
    "client_id": "GOOGLE_CLIENT_ID",
    "auth_uri": "GOOGLE_AUTH_URI",
    "token_uri": "GOOGLE_TOKEN_URI",
    "auth_provider_x509_cert_url": "GOOGLE_AUTH_PROVIDER_X509_CERT_URL",
    "client_x509_cert_url": "GOOGLE_CLIENT_X509_CERT_URL",
    "universe_domain": "GOOGLE_UNIVERSE_DOMAIN",
}


def quote_env(value: str) -> str:
    escaped = value.replace("\\", "\\\\").replace("\n", "\\n").replace('"', '\\"')
    return f'"{escaped}"'


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Convert credentials.json service-account keys to .env format."
    )
    parser.add_argument(
        "--input",
        default="credentials.json",
        help="Path to credentials JSON file (default: credentials.json)",
    )
    parser.add_argument(
        "--output",
        default=".env",
        help="Path to output env file (default: .env)",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Overwrite output file if it already exists",
    )
    args = parser.parse_args()

    input_path = Path(args.input)
    output_path = Path(args.output)

    if not input_path.exists():
        raise SystemExit(f"input file does not exist: {input_path}")
    if output_path.exists() and not args.force:
        raise SystemExit(
            f"output file already exists: {output_path} (use --force to overwrite)"
        )

    with input_path.open("r", encoding="utf-8") as handle:
        credentials = json.load(handle)

    lines = ["# Generated from credentials.json"]
    for source_key, env_key in KEY_MAP.items():
        value = str(credentials.get(source_key, ""))
        lines.append(f"{env_key}={quote_env(value)}")

    if "API_SECRET" not in credentials:
        lines.append("API_SECRET=replace-with-a-long-random-secret")

    output_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"Wrote {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
