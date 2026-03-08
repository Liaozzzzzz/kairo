---
name: "douyin-uploader"
description: "Fully automated video publishing to Douyin. Requires valid cookies for unattended operation."
---

# Douyin Uploader Skill

This skill automates the entire process of uploading and publishing videos to Douyin (TikTok China) without user intervention.

## Usage

`openclaw agent --agent main --message "Use douyin-uploader to upload <video_path> [options]"`

## Prerequisites

1.  **Authentication**: Valid cookies (`cookies.json` or equivalent) MUST be provided or pre-saved in the browser profile.
2.  **Video File**: Local path to the video.

## Input Parameters

- **Video Path** (Required): Path to video file.
- **Title** (Optional): Defaults to filename.
- **Description** (Optional): Defaults to empty or title.
- **Tags** (Optional): Space/comma separated.
- **Cookie Path** (Optional): Path to a JSON file containing cookies. If not provided, the agent should attempt to use the default browser context or a known default cookie location.

## Automation Workflow

### 1. Initialization & Login Verification (Zero-Touch)

- **Context**: Launch browser (persistent context recommended).
- **Cookies**:
  - If `Cookie Path` is provided, load it.
  - Navigate to `https://creator.douyin.com/creator-micro/content/upload`.
- **Verification**:
  - Check if redirected to login page.
  - **CRITICAL**: If not logged in, the automation CANNOT proceed unattended.
  - _Action_: If login fails, throw a fatal error: "Authentication failed. Please provide valid cookies or log in manually once to save session." (Do NOT wait for QR code unless explicitly interactive).

### 2. Upload & Metadata

- **Upload**:
  - Wait for file input selector.
  - Set input files to **Video Path**.
  - Wait for upload progress to reach 100% (monitor text "Upload Success" or similar).
- **Fill Metadata**:
  - **Title/Description**: Locate the editor. Clear existing text if any. Type the **Title** and **Description**.
  - **Tags**: Enter tags one by one, pressing `Enter` after each.
- **Location/Permissions**:
  - Set permissions to "Public" (default).
  - Set location if requested (otherwise skip).

### 3. Cover Selection (Auto-Resolve)

- **Detection**: Monitor for "Set Cover" requirement.
- **Action**:
  - Click "Select Cover" (or similar button).
  - **Crucial**: Immediately select the _first_ generated thumbnail/frame.
  - Click "Complete" / "Confirm" / "Crop" button in the modal to close it.
  - _Goal_: Satisfy the cover requirement with the default option to avoid blocking.

### 4. Publishing (Auto-Confirm)

- **Action**: Click the "Publish" button.
- **Popups**:
  - Watch for "Confirm Publish" or "Sync to Toutiao" dialogs.
  - Automatically click "Publish" / "Confirm" / "Continue".
- **Success State**:
  - Wait for the URL to change to the management page OR for a "Publish Successful" toast/modal.
  - Take a screenshot of the success state if possible.

## Error Recovery

- **Upload Stuck**: If upload stays at 0% for >30s, refresh and retry.
- **Modal Blocking**: If any unknown modal appears, attempt to close it (Esc or "X" button) or click the primary positive action.
