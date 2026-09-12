import { createSettingsRuntimeModule } from "./settings_runtime.js";
import { createSettingsProfilesModule } from "./settings_profiles.js";
import { createSettingsOpsModule } from "./settings_ops.js";

export function createSettingsModule({ state, actions, escapeHtml, populateKnownModelList }) {
  const runtime = createSettingsRuntimeModule({ state, actions, escapeHtml, populateKnownModelList });
  Object.assign(actions, runtime);

  const profiles = createSettingsProfilesModule({ state, actions, escapeHtml, populateKnownModelList });
  Object.assign(actions, profiles);

  const ops = createSettingsOpsModule({ state, actions, escapeHtml });
  Object.assign(actions, ops);

  let bindingsInitialized = false;

  function initSettingsBindings() {
    if (bindingsInitialized) return;
    bindingsInitialized = true;

    document.getElementById("btn-settings-tab-standard")?.addEventListener("click", () => runtime.switchSettingsTab("standard"));
    document.getElementById("btn-settings-tab-providers")?.addEventListener("click", () => runtime.switchSettingsTab("providers"));
    document.getElementById("settings-manage-profiles-btn")?.addEventListener("click", () => runtime.switchSettingsTab("providers"));
    document.getElementById("settings-close-btn")?.addEventListener("click", runtime.closeSettingsModal);
    document.getElementById("settings-cancel-btn")?.addEventListener("click", runtime.closeSettingsModal);
    document.getElementById("settings-form")?.addEventListener("submit", runtime.saveSettings);

    document.getElementById("input-provider-profile")?.addEventListener("change", (e) => profiles.onChatProfileChange(e.target.value));
    document.getElementById("input-provider")?.addEventListener("change", (e) => runtime.suggestBaseURL(e.target.value));
    document.getElementById("input-compact-profile")?.addEventListener("change", (e) => profiles.onCompactProfileChange(e.target.value));
    document.getElementById("input-compact-provider")?.addEventListener("change", (e) => runtime.suggestCompactBaseURL(e.target.value));

    document.getElementById("settings-add-ignore-btn")?.addEventListener("click", ops.addIgnorePattern);
    document.getElementById("settings-add-collapse-btn")?.addEventListener("click", ops.addCollapsePattern);
    document.getElementById("settings-add-mcp-btn")?.addEventListener("click", ops.addMCPServer);

    document.getElementById("settings-new-provider-btn")?.addEventListener("click", profiles.newProviderForm);
    document.getElementById("active-chat-profile")?.addEventListener("change", (e) => profiles.setActiveProfile(e.target.value, "chat"));
    document.getElementById("active-compaction-profile")?.addEventListener("change", (e) => profiles.setActiveProfile(e.target.value, "compaction"));
    document.getElementById("provider-close-btn")?.addEventListener("click", profiles.closeProviderForm);
    document.getElementById("provider-cancel-btn")?.addEventListener("click", profiles.closeProviderForm);
    document.getElementById("pe-provider")?.addEventListener("change", (e) => profiles.suggestProviderBaseUrl(e.target.value));
    document.getElementById("settings-save-provider-btn")?.addEventListener("click", profiles.saveProvider);

    document.getElementById("ignore-chips-list")?.addEventListener("click", (e) => {
      const btn = e.target.closest("[data-remove-ignore-index]");
      if (!btn) return;
      ops.removeIgnorePattern(Number(btn.getAttribute("data-remove-ignore-index")));
    });

    document.getElementById("collapse-chips-list")?.addEventListener("click", (e) => {
      const btn = e.target.closest("[data-remove-collapse-index]");
      if (!btn) return;
      ops.removeCollapsePattern(Number(btn.getAttribute("data-remove-collapse-index")));
    });

    document.getElementById("snapshots-list")?.addEventListener("click", (e) => {
      const revertBtn = e.target.closest("[data-revert-snapshot]");
      if (revertBtn) {
        ops.revertToSnapshot(revertBtn.getAttribute("data-revert-snapshot"));
        return;
      }
      const deleteBtn = e.target.closest("[data-delete-snapshot]");
      if (deleteBtn) {
        ops.deleteSnapshot(deleteBtn.getAttribute("data-delete-snapshot"));
      }
    });

    document.getElementById("mcp-servers-list")?.addEventListener("click", (e) => {
      const btn = e.target.closest("[data-delete-mcp]");
      if (!btn) return;
      ops.deleteMCPServer(btn.getAttribute("data-delete-mcp"));
    });

    document.getElementById("providers-list")?.addEventListener("click", (e) => {
      const editBtn = e.target.closest("[data-edit-provider]");
      if (editBtn) {
        profiles.editProvider(editBtn.getAttribute("data-edit-provider"));
        return;
      }
      const deleteBtn = e.target.closest("[data-delete-provider]");
      if (deleteBtn) {
        profiles.deleteProvider(deleteBtn.getAttribute("data-delete-provider"));
      }
    });
  }

  return {
    ...runtime,
    ...profiles,
    ...ops,
    initSettingsBindings,
  };
}
