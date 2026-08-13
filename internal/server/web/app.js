document.addEventListener('DOMContentLoaded', () => {
  // State
  let projects = [];
  let deployments = [];
  let webhooks = [];
  let autoRefreshInterval = null;
  let liveTickerInterval = null;
  let isAutoRefreshOn = true;
  let activeTerminalDepId = null;

  // Header & Controls Elements
  const btnNewProject = document.getElementById('btnNewProject');
  const btnRefresh = document.getElementById('btnRefresh');
  const btnClear = document.getElementById('btnClear');
  const autoRefreshToggle = document.getElementById('autoRefreshToggle');
  const liveDot = document.getElementById('liveDot');
  const liveStatus = document.getElementById('liveStatus');
  const searchInput = document.getElementById('searchInput');
  const eventFilter = document.getElementById('eventFilter');

  // Navigation Tabs
  const navTabs = document.getElementById('navTabs');
  const tabBtns = document.querySelectorAll('.tab-btn');
  const tabContents = document.querySelectorAll('.tab-content');

  // Stats Elements
  const statProjects = document.getElementById('statProjects');
  const statProjectsTotal = document.getElementById('statProjectsTotal');
  const statDeployments = document.getElementById('statDeployments');
  const statSuccess = document.getElementById('statSuccess');
  const statSuccessRate = document.getElementById('statSuccessRate');
  const statFailed = document.getElementById('statFailed');
  const badgeProjects = document.getElementById('badgeProjects');
  const badgeDeployments = document.getElementById('badgeDeployments');
  const badgeWebhooks = document.getElementById('badgeWebhooks');

  // Containers
  const projectsGrid = document.getElementById('projectsGrid');
  const emptyProjects = document.getElementById('emptyProjects');
  const deploymentsTbody = document.getElementById('deploymentsTbody');
  const emptyDeployments = document.getElementById('emptyDeployments');
  const payloadTbody = document.getElementById('payloadTbody');
  const emptyState = document.getElementById('emptyState');

  // Modals
  const projectModal = document.getElementById('projectModal');
  const projectForm = document.getElementById('projectForm');
  const modalProjectTitle = document.getElementById('modalProjectTitle');
  const btnCloseProjectModal = document.getElementById('btnCloseProjectModal');
  const btnCancelProjectModal = document.getElementById('btnCancelProjectModal');

  const terminalModal = document.getElementById('terminalModal');
  const terminalTitle = document.getElementById('terminalTitle');
  const terminalContent = document.getElementById('terminalContent');
  const btnCopyLog = document.getElementById('btnCopyLog');
  const btnCloseTerminalModal = document.getElementById('btnCloseTerminalModal');

  // Failure Modal
  const failureModal = document.getElementById('failureModal');
  const failureContent = document.getElementById('failureContent');
  const btnCloseFailureModal = document.getElementById('btnCloseFailureModal');
  const btnCopyFailureLog = document.getElementById('btnCopyFailureLog');
  const btnRetryDeployment = document.getElementById('btnRetryDeployment');
  const failMetaProject = document.getElementById('failMetaProject');
  const failMetaRepo = document.getElementById('failMetaRepo');
  const failMetaTrigger = document.getElementById('failMetaTrigger');
  const failMetaTime = document.getElementById('failMetaTime');
  let currentFailedProjectId = null;

  const toast = document.getElementById('toast');

  // --- Initialize ---
  init();

  function init() {
    setupEventListeners();
    fetchAllData();
    startAutoRefresh();
    startLiveTicker();
  }

  function setupEventListeners() {
    // 1. Tab Switching
    if (navTabs) {
      navTabs.addEventListener('click', (e) => {
        const btn = e.target.closest('.tab-btn');
        if (!btn) return;
        const targetTab = btn.getAttribute('data-tab');
        
        tabBtns.forEach(b => b.classList.remove('active'));
        tabContents.forEach(c => c.classList.remove('active'));

        btn.classList.add('active');
        const contentEl = document.getElementById(targetTab);
        if (contentEl) contentEl.classList.add('active');
      });
    }

    // 2. Header Actions
    if (btnNewProject) {
      btnNewProject.addEventListener('click', () => openProjectModal());
    }

    if (btnRefresh) {
      btnRefresh.addEventListener('click', () => {
        const icon = btnRefresh.querySelector('.refresh-icon');
        if (icon) icon.classList.add('spin');
        fetchAllData().finally(() => {
          setTimeout(() => {
            if (icon) icon.classList.remove('spin');
          }, 600);
        });
      });
    }

    if (btnClear) {
      btnClear.addEventListener('click', clearWebhooks);
    }

    if (autoRefreshToggle) {
      autoRefreshToggle.addEventListener('change', (e) => {
        isAutoRefreshOn = e.target.checked;
        if (isAutoRefreshOn) {
          if (liveDot) liveDot.classList.remove('paused');
          if (liveStatus) liveStatus.textContent = 'Auto-refresh On';
          startAutoRefresh();
        } else {
          if (liveDot) liveDot.classList.add('paused');
          if (liveStatus) liveStatus.textContent = 'Paused';
          stopAutoRefresh();
        }
      });
    }

    // 3. Search and Filter
    if (searchInput) {
      searchInput.addEventListener('input', renderWebhooks);
    }
    if (eventFilter) {
      eventFilter.addEventListener('change', renderWebhooks);
    }

    // 4. EVENT DELEGATION: Projects Grid
    if (projectsGrid) {
      projectsGrid.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-action]');
        if (!btn) return;
        const action = btn.dataset.action;
        const id = btn.dataset.id;

        if (action === 'edit-project') {
          editProject(id);
        } else if (action === 'delete-project') {
          deleteProject(id);
        } else if (action === 'deploy-project') {
          triggerDeploy(id, btn);
        } else if (action === 'copy-path') {
          copyText(btn.dataset.path || btn.textContent.trim());
        }
      });
    }

    // 5. EVENT DELEGATION: Deployment Logs Table
    if (deploymentsTbody) {
      deploymentsTbody.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-action]');
        if (!btn) return;
        const action = btn.dataset.action;
        const id = btn.dataset.id;

        if (action === 'view-log') {
          openTerminalModal(id);
        } else if (action === 'view-failure') {
          openFailureModal(id);
        }
      });
    }

    // 6. EVENT DELEGATION: Webhook Payloads Table
    if (payloadTbody) {
      payloadTbody.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-action]');
        if (!btn) return;
        const action = btn.dataset.action;

        if (action === 'copy-id') {
          copyText(btn.dataset.text || btn.textContent.trim());
        } else if (action === 'quick-add-project') {
          const repo = btn.dataset.repo;
          openProjectModal({
            name: repo ? repo.split('/').pop() : 'New Project',
            repository: repo || '',
            branch: 'main',
            projectPath: '',
            script: 'cd /path/to/project\ngit pull origin main\ndocker compose down\ndocker compose up --build -d',
            enabled: true
          });
        }
      });
    }

    // 7. Modals Listeners
    if (btnCloseProjectModal) btnCloseProjectModal.addEventListener('click', closeProjectModal);
    if (btnCancelProjectModal) btnCancelProjectModal.addEventListener('click', closeProjectModal);
    if (projectForm) projectForm.addEventListener('submit', saveProject);

    if (btnCloseTerminalModal) btnCloseTerminalModal.addEventListener('click', closeTerminalModal);
    if (btnCopyLog) {
      btnCopyLog.addEventListener('click', () => {
        if (terminalContent) {
          navigator.clipboard.writeText(terminalContent.textContent);
          showToast('Console output log copied to clipboard!');
        }
      });
    }

    if (btnCloseFailureModal) btnCloseFailureModal.addEventListener('click', closeFailureModal);
    if (btnCopyFailureLog) {
      btnCopyFailureLog.addEventListener('click', () => {
        if (failureContent) {
          navigator.clipboard.writeText(failureContent.textContent);
          showToast('Failure log output copied to clipboard!');
        }
      });
    }
    if (btnRetryDeployment) {
      btnRetryDeployment.addEventListener('click', () => {
        if (currentFailedProjectId) {
          closeFailureModal();
          triggerDeploy(currentFailedProjectId);
        }
      });
    }

    // Close modals on overlay click or Escape key
    window.addEventListener('click', (e) => {
      if (e.target === projectModal) closeProjectModal();
      if (e.target === terminalModal) closeTerminalModal();
      if (e.target === failureModal) closeFailureModal();
    });

    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        closeProjectModal();
        closeTerminalModal();
        closeFailureModal();
      }
    });
  }

  // --- Auto-Refresh & Live Ticker ---
  function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshInterval = setInterval(() => {
      if (isAutoRefreshOn) {
        fetchAllData(true);
      }
    }, 2500);
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) {
      clearInterval(autoRefreshInterval);
      autoRefreshInterval = null;
    }
  }

  function startLiveTicker() {
    if (liveTickerInterval) clearInterval(liveTickerInterval);
    liveTickerInterval = setInterval(() => {
      const hasRunning = deployments.some(d => d.status === 'RUNNING');
      if (hasRunning) {
        renderDeployments();
      }
      // Streaming update terminal content if terminal modal is open for a running deployment
      if (activeTerminalDepId) {
        const activeDep = deployments.find(d => d.id === activeTerminalDepId);
        if (activeDep && activeDep.status === 'RUNNING') {
          fetchAndStreamTerminal(activeTerminalDepId);
        }
      }
    }, 1000);
  }

  // --- Data Fetching ---
  async function fetchAllData(silent = false) {
    await Promise.all([
      fetchProjects(silent),
      fetchDeployments(silent),
      fetchWebhooks(silent)
    ]);
  }

  async function fetchProjects(silent = false) {
    try {
      const res = await fetch('/api/projects');
      if (!res.ok) throw new Error('Failed to load projects');
      const data = await res.json();
      projects = data.projects || [];
      renderProjects();
      updateStats();
    } catch (err) {
      if (!silent) showToast(err.message, true);
    }
  }

  async function fetchDeployments(silent = false) {
    try {
      const res = await fetch('/api/deployments');
      if (!res.ok) throw new Error('Failed to load deployments');
      const data = await res.json();
      deployments = data.deployments || [];
      renderDeployments();
      updateStats();
    } catch (err) {
      if (!silent) showToast(err.message, true);
    }
  }

  async function fetchWebhooks(silent = false) {
    try {
      const res = await fetch('/api/webhooks');
      if (!res.ok) throw new Error('Failed to load webhooks');
      const data = await res.json();
      webhooks = data.events || [];
      renderWebhooks();
    } catch (err) {
      if (!silent) showToast(err.message, true);
    }
  }

  // --- Rendering Projects Grid ---
  function renderProjects() {
    if (badgeProjects) badgeProjects.textContent = projects.length;
    if (projects.length === 0) {
      if (projectsGrid) projectsGrid.innerHTML = '';
      if (emptyProjects) emptyProjects.classList.remove('hidden');
      return;
    }

    if (emptyProjects) emptyProjects.classList.add('hidden');
    if (projectsGrid) {
      projectsGrid.innerHTML = projects.map(p => `
        <div class="project-card" data-project-id="${escapeHTML(p.id)}">
          <div class="project-card-header">
            <div class="project-name-wrap">
              <span class="project-name">${escapeHTML(p.name)}</span>
              <span class="project-repo">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4"></path>
                </svg>
                ${escapeHTML(p.repository)}
              </span>
            </div>
            <span class="badge ${p.enabled ? 'badge-status-success' : 'badge-status-failed'}">
              ${p.enabled ? 'Enabled' : 'Disabled'}
            </span>
          </div>

          <div class="project-details">
            <div class="project-meta-row">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="6" y1="3" x2="6" y2="15"></line>
                <circle cx="18" cy="6" r="3"></circle>
                <circle cx="6" cy="18" r="3"></circle>
                <path d="M18 9a9 9 0 0 1-9 9"></path>
              </svg>
              <span>Target Branch:</span>
              <span class="branch-badge">${escapeHTML(p.branch)}</span>
            </div>

            <div class="project-meta-row">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
              </svg>
              <span>Server Path:</span>
              <span class="path-code" data-action="copy-path" data-path="${escapeHTML(p.projectPath)}" title="Click to copy path">${escapeHTML(p.projectPath)}</span>
            </div>

            <div class="script-box">${escapeHTML(p.script || '# No script provided')}</div>
          </div>

          <div class="project-card-footer">
            <div class="project-actions">
              <button type="button" class="btn btn-xs btn-secondary" data-action="edit-project" data-id="${escapeHTML(p.id)}">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                </svg>
                Edit
              </button>
              <button type="button" class="btn btn-xs btn-danger-outline" data-action="delete-project" data-id="${escapeHTML(p.id)}">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
                Delete
              </button>
            </div>
            <button type="button" class="btn btn-xs btn-success-outline" data-action="deploy-project" data-id="${escapeHTML(p.id)}">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="5 3 19 12 5 21 5 3"></polygon>
              </svg>
              <span>Deploy Now</span>
            </button>
          </div>
        </div>
      `).join('');
    }
  }

  // --- Rendering Deployments ---
  function renderDeployments() {
    if (badgeDeployments) badgeDeployments.textContent = deployments.length;
    if (deployments.length === 0) {
      if (deploymentsTbody) deploymentsTbody.innerHTML = '';
      if (emptyDeployments) emptyDeployments.classList.remove('hidden');
      return;
    }

    if (emptyDeployments) emptyDeployments.classList.add('hidden');
    if (deploymentsTbody) {
      deploymentsTbody.innerHTML = deployments.map(d => {
        const isRunning = d.status === 'RUNNING';
        const isFailed = d.status === 'FAILED';

        return `
          <tr>
            <td>
              ${isRunning ? `
                <span class="badge badge-status-running">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spin" style="margin-right:4px; vertical-align:middle;">
                    <circle cx="12" cy="12" r="10"></circle>
                    <path d="M12 2a10 10 0 0 1 10 10"></path>
                  </svg>
                  RUNNING
                </span>
              ` : (isFailed ? `
                <span class="badge badge-status-failed clickable-badge" data-action="view-failure" data-id="${escapeHTML(d.id)}" title="Click to view failure details">
                  FAILED ⚠️
                </span>
              ` : `
                <span class="badge badge-status-success">${escapeHTML(d.status)}</span>
              `)}
            </td>
            <td style="font-weight:700; color:white;">${escapeHTML(d.projectName || d.projectId)}</td>
            <td><span style="font-family:var(--font-mono); color:var(--accent-indigo);">${escapeHTML(d.repository)}</span></td>
            <td><span class="branch-badge">${escapeHTML(d.branch)}</span></td>
            <td><span class="sender-pill">${escapeHTML(d.triggeredBy || 'Manual')}</span></td>
            <td>
              <span style="font-family:var(--font-mono); font-weight:${isRunning ? 'bold' : 'normal'}; color:${isRunning ? '#38bdf8' : 'inherit'};">
                ${formatDuration(d.durationMs, d.status, d.startedAt)}
              </span>
            </td>
            <td><span class="time-stamp">${formatTime(d.startedAt)}</span></td>
            <td>
              <div style="display:flex; gap:0.4rem; flex-wrap:wrap;">
                <button type="button" class="btn btn-xs btn-secondary" data-action="view-log" data-id="${escapeHTML(d.id)}">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="4 17 10 11 16 17 20 13"></polyline>
                    <line x1="4" y1="7" x2="20" y2="7"></line>
                  </svg>
                  View Log
                </button>
                ${isFailed ? `
                  <button type="button" class="btn btn-xs btn-danger-outline" data-action="view-failure" data-id="${escapeHTML(d.id)}" title="Click to view error diagnosis">
                    Why Failed?
                  </button>
                ` : ''}
              </div>
            </td>
          </tr>
        `;
      }).join('');
    }
  }

  // --- Rendering Webhook Payloads ---
  function renderWebhooks() {
    const query = (searchInput ? searchInput.value : '').toLowerCase().trim();
    const filter = eventFilter ? eventFilter.value : 'ALL';

    const filtered = webhooks.filter(ev => {
      const matchesFilter = filter === 'ALL' || ev.eventType === filter;
      const matchesSearch = !query || 
        (ev.deliveryID && ev.deliveryID.toLowerCase().includes(query)) ||
        (ev.eventType && ev.eventType.toLowerCase().includes(query)) ||
        (ev.repositoryName && ev.repositoryName.toLowerCase().includes(query)) ||
        (ev.prTitle && ev.prTitle.toLowerCase().includes(query)) ||
        (ev.sender && ev.sender.toLowerCase().includes(query));

      return matchesFilter && matchesSearch;
    });

    if (badgeWebhooks) badgeWebhooks.textContent = filtered.length;

    if (filtered.length === 0) {
      if (payloadTbody) payloadTbody.innerHTML = '';
      if (emptyState) emptyState.classList.remove('hidden');
      return;
    }

    if (emptyState) emptyState.classList.add('hidden');
    if (payloadTbody) {
      payloadTbody.innerHTML = filtered.map(ev => `
        <tr>
          <td>
            <span class="delivery-id" data-action="copy-id" data-text="${escapeHTML(ev.deliveryID)}" title="Click to copy delivery ID">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
              </svg>
              ${escapeHTML(ev.deliveryID)}
            </span>
          </td>
          <td><span class="badge badge-event-${escapeHTML(ev.eventType)}">${escapeHTML(ev.eventType)}</span></td>
          <td>
            <div style="display:inline-flex; align-items:center; gap:0.5rem;">
              <span style="font-family:var(--font-mono); color:var(--text-primary); font-weight:600;">${escapeHTML(ev.repositoryName || '-')}</span>
              ${ev.repositoryName ? `
                <button type="button" class="btn btn-xs btn-primary" data-action="quick-add-project" data-repo="${escapeHTML(ev.repositoryName)}" title="Add project for this repo">
                  + Add Project
                </button>
              ` : ''}
            </div>
          </td>
          <td>${ev.action ? `<span class="badge badge-status-running">${escapeHTML(ev.action)}</span>` : '<span style="color:var(--text-muted);">-</span>'}</td>
          <td>${ev.prNumber ? `<span class="branch-badge">#${ev.prNumber}</span>` : '<span style="color:var(--text-muted);">-</span>'}</td>
          <td>${ev.prTitle ? `<div style="max-width:220px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title="${escapeHTML(ev.prTitle)}">${escapeHTML(ev.prTitle)}</div>` : '<span style="color:var(--text-muted);">-</span>'}</td>
          <td>
            <span class="sender-pill">
              <span class="sender-avatar">${(ev.sender || 'U').charAt(0).toUpperCase()}</span>
              ${escapeHTML(ev.sender || 'Unknown')}
            </span>
          </td>
          <td><span class="time-stamp">${formatTime(ev.receivedAt)}</span></td>
        </tr>
      `).join('');
    }
  }

  // --- Update Stats Dashboard ---
  function updateStats() {
    const activeCount = projects.filter(p => p.enabled).length;
    if (statProjects) statProjects.textContent = activeCount;
    if (statProjectsTotal) statProjectsTotal.textContent = `of ${projects.length} configured`;
    if (statDeployments) statDeployments.textContent = deployments.length;

    const successCount = deployments.filter(d => d.status === 'SUCCESS').length;
    const failedCount = deployments.filter(d => d.status === 'FAILED').length;

    if (statSuccess) statSuccess.textContent = successCount;
    if (statFailed) statFailed.textContent = failedCount;

    if (statSuccessRate) {
      const rate = deployments.length > 0 ? Math.round((successCount / deployments.length) * 100) : 0;
      statSuccessRate.textContent = `${rate}% success rate`;
    }
  }

  // --- Actions ---
  function editProject(id) {
    const proj = projects.find(p => p.id === id);
    if (proj) openProjectModal(proj);
  }

  function openProjectModal(proj = null) {
    document.getElementById('projectId').value = proj ? proj.id : '';
    document.getElementById('projectName').value = proj ? proj.name : '';
    document.getElementById('projectRepo').value = proj ? proj.repository : '';
    document.getElementById('projectBranch').value = proj ? proj.branch : 'main';
    document.getElementById('projectPath').value = proj ? proj.projectPath : '';
    document.getElementById('projectScript').value = proj ? proj.script : 'cd /path/to/project\ngit pull origin main\ndocker compose down\ndocker compose up --build -d';
    document.getElementById('projectEnabled').checked = proj ? proj.enabled : true;

    if (modalProjectTitle) modalProjectTitle.textContent = proj ? 'Edit Deployment Project' : 'Add Deployment Project';
    if (projectModal) projectModal.classList.remove('hidden');
  }

  function closeProjectModal() {
    if (projectModal) projectModal.classList.add('hidden');
    if (projectForm) projectForm.reset();
  }

  async function saveProject(e) {
    e.preventDefault();
    const id = document.getElementById('projectId').value;
    const body = {
      name: document.getElementById('projectName').value,
      repository: document.getElementById('projectRepo').value,
      branch: document.getElementById('projectBranch').value,
      projectPath: document.getElementById('projectPath').value,
      script: document.getElementById('projectScript').value,
      enabled: document.getElementById('projectEnabled').checked
    };

    const saveBtn = document.getElementById('btnSaveProject');
    const origHTML = saveBtn ? saveBtn.innerHTML : '';
    if (saveBtn) {
      saveBtn.disabled = true;
      saveBtn.innerHTML = `
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spin">
          <circle cx="12" cy="12" r="10"></circle>
          <path d="M12 2a10 10 0 0 1 10 10"></path>
        </svg>
        Saving...
      `;
    }

    try {
      const url = id ? `/api/projects/${id}` : '/api/projects';
      const method = id ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });

      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Failed to save project');
      }
      showToast(id ? 'Project updated successfully!' : 'Project created successfully!');
      closeProjectModal();
      fetchProjects();
    } catch (err) {
      showToast(err.message, true);
    } finally {
      if (saveBtn) {
        saveBtn.disabled = false;
        saveBtn.innerHTML = origHTML;
      }
    }
  }

  async function deleteProject(id) {
    if (!confirm('Are you sure you want to delete this project deployment config?')) return;
    try {
      const res = await fetch(`/api/projects/${id}`, { method: 'DELETE' });
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Failed to delete project');
      }
      showToast('Project deleted successfully');
      fetchProjects();
    } catch (err) {
      showToast(err.message, true);
    }
  }

  async function triggerDeploy(id, buttonEl = null) {
    let origHTML = '';
    if (buttonEl) {
      origHTML = buttonEl.innerHTML;
      buttonEl.disabled = true;
      buttonEl.innerHTML = `
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spin">
          <circle cx="12" cy="12" r="10"></circle>
          <path d="M12 2a10 10 0 0 1 10 10"></path>
        </svg>
        Deploying...
      `;
    }

    try {
      showToast('Triggering manual deployment...');
      const res = await fetch(`/api/projects/${id}/deploy`, { method: 'POST' });
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Failed to trigger deployment');
      }
      const data = await res.json();
      showToast('Deployment launched successfully!');

      // Switch to Deployments tab
      const tabBtn = document.querySelector('[data-tab="deploymentsTab"]');
      if (tabBtn) tabBtn.click();
      
      await fetchDeployments();

      if (data.deployment && data.deployment.id) {
        setTimeout(() => openTerminalModal(data.deployment.id), 400);
      }
    } catch (err) {
      showToast(err.message, true);
    } finally {
      if (buttonEl) {
        buttonEl.disabled = false;
        buttonEl.innerHTML = origHTML;
      }
    }
  }

  async function openTerminalModal(depId) {
    activeTerminalDepId = depId;
    if (terminalTitle) terminalTitle.textContent = `Deployment Log: ${depId}`;
    if (terminalContent) terminalContent.textContent = 'Loading execution log output...';
    if (terminalModal) terminalModal.classList.remove('hidden');

    await fetchAndStreamTerminal(depId);
  }

  async function fetchAndStreamTerminal(depId) {
    try {
      const res = await fetch(`/api/deployments/${depId}`);
      if (!res.ok) throw new Error('Log record not found');
      const data = await res.json();
      const dep = data.deployment;
      if (terminalTitle) terminalTitle.textContent = `Console Log — ${dep.projectName || dep.projectId} (${dep.repository}:${dep.branch})`;
      if (terminalContent) {
        terminalContent.textContent = dep.log || 'No log output recorded.';
        // Auto scroll to bottom of log output
        terminalContent.scrollTop = terminalContent.scrollHeight;
      }
    } catch (err) {
      if (terminalContent) terminalContent.textContent = `Failed to load execution log: ${err.message}`;
    }
  }

  function closeTerminalModal() {
    if (terminalModal) terminalModal.classList.add('hidden');
    activeTerminalDepId = null;
  }

  async function openFailureModal(depId) {
    if (failureContent) failureContent.textContent = 'Loading failure diagnosis details...';
    if (failMetaProject) failMetaProject.textContent = 'Loading...';
    if (failMetaRepo) failMetaRepo.textContent = '-';
    if (failMetaTrigger) failMetaTrigger.textContent = '-';
    if (failMetaTime) failMetaTime.textContent = '-';
    if (failureModal) failureModal.classList.remove('hidden');

    try {
      const res = await fetch(`/api/deployments/${depId}`);
      if (!res.ok) throw new Error('Deployment log record not found');
      const data = await res.json();
      const dep = data.deployment;

      currentFailedProjectId = dep.projectId;
      if (failMetaProject) failMetaProject.textContent = dep.projectName || dep.projectId;
      if (failMetaRepo) failMetaRepo.textContent = `${dep.repository}:${dep.branch}`;
      if (failMetaTrigger) failMetaTrigger.textContent = dep.triggeredBy || 'Manual';
      if (failMetaTime) failMetaTime.textContent = `${formatTime(dep.startedAt)} (${formatDuration(dep.durationMs, dep.status, dep.startedAt)})`;
      if (failureContent) failureContent.textContent = dep.log || 'No detailed log recorded for this execution.';
    } catch (err) {
      if (failureContent) failureContent.textContent = `Failed to load failure record details: ${err.message}`;
    }
  }

  function closeFailureModal() {
    if (failureModal) failureModal.classList.add('hidden');
    currentFailedProjectId = null;
  }

  async function clearWebhooks() {
    if (!confirm('Clear all recorded webhook payload history?')) return;
    try {
      const res = await fetch('/api/webhooks/clear', { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to clear webhooks');
      showToast('Webhook history cleared successfully');
      fetchWebhooks();
    } catch (err) {
      showToast(err.message, true);
    }
  }

  // --- Toast Helpers ---
  function showToast(msg, isError = false) {
    if (!toast) return;
    toast.textContent = msg;
    toast.style.borderColor = isError ? 'var(--accent-rose)' : 'var(--accent-indigo)';
    toast.classList.remove('hidden');
    setTimeout(() => {
      toast.classList.add('hidden');
    }, 3200);
  }

  function copyText(text) {
    if (!text) return;
    navigator.clipboard.writeText(text);
    showToast('Copied to clipboard!');
  }

  function formatTime(isoStr) {
    if (!isoStr) return '-';
    try {
      const d = new Date(isoStr);
      return d.toLocaleString(undefined, {
        month: 'short', day: 'numeric',
        hour: '2-digit', minute: '2-digit', second: '2-digit'
      });
    } catch (e) {
      return isoStr;
    }
  }

  function formatDuration(ms, status, startedAt) {
    if (status === 'RUNNING' && startedAt) {
      const startMs = new Date(startedAt).getTime();
      const nowMs = Date.now();
      const elapsedSec = Math.max(0, Math.floor((nowMs - startMs) / 1000));
      return `${elapsedSec}s running...`;
    }
    if (!ms) return '0s';
    if (ms < 1000) return `${ms}ms`;
    const sec = (ms / 1000).toFixed(1);
    return `${sec}s`;
  }

  function escapeHTML(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }
});
