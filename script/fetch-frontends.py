import os
import io
import zipfile
import requests
import yaml
import shutil


def download_and_extract_merge(repository, version, target_dir):
    print(f"Downloading from repository: {repository}, version: {version}")

    download_url = f"{repository}/releases/download/{version}/dist.zip"
    response = requests.get(download_url)
    response.raise_for_status()

    zip_bytes = io.BytesIO(response.content)
    with zipfile.ZipFile(zip_bytes) as zip_ref:
        for file_info in zip_ref.infolist():
            if file_info.filename.startswith("dist/"):
                extracted_path = os.path.join(
                    target_dir, file_info.filename[5:])
                if file_info.filename.endswith('/'):
                    os.makedirs(extracted_path, exist_ok=True)
                else:
                    os.makedirs(os.path.dirname(extracted_path), exist_ok=True)
                    with zip_ref.open(file_info) as source, open(extracted_path, "wb") as target:
                        shutil.copyfileobj(source, target)


def main():
    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    templates_file = os.path.join(
        root_dir, "service", "singleton", "frontend-templates.yaml")

    try:
        with open(templates_file, "r") as f:
            templates = yaml.safe_load(f)
    except FileNotFoundError:
        print(f"Error: {templates_file} not found.")
        return
    except yaml.YAMLError as e:
        print(f"Error: Invalid YAML in {templates_file}: {e}")
        return

    if templates:
        for template in templates:
            path = template.get("path")
            repository = template.get("repository")
            version = template.get("version")

            if path and repository and version:
                target_dir = os.path.join(root_dir, "cmd", "dashboard", path)
                download_and_extract_merge(repository, version, target_dir)


if __name__ == "__main__":
    main()
