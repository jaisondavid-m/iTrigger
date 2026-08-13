document.addEventListener('DOMContentLoaded', () => {
  // State
  let projects = [];
  let deployments = [];
  let webhooks = [];
  let autoRefreshInterval = null;
  let isAutoRefreshOn = true;

  // DOM Elements
  const btnNewProject = document.getElementById('btnNewProject');
  const btnRefresh = document.getElementById('btnRefresh');
  const btnClear = document.getElementById('btnClear');
  const autoRefreshToggle = document.getElementById('autoRefreshToggle');
  const liveDot = document.getElementById('liveDot');
  const liveStatus = document.getElementById('liveStatus');
  const searchInput = document.getElementById('searchInput');
  const eventFilter = document.getElementById('eventFilter');

  // Tabs
  const tabBtns = document.querySelectorAll('.tab-btn');
  const tabContents = document.querySelectorAll('.tab-content');

  // Stats
  const statProjects = document.getElementById('statProjects');
  const statDeployments = document.getElementById('statDeployments');
  const statSuccess = document.getElementById('statSuccess');
  const statFailed = document.getElementById('statFailed');
  const badgeProjects = document.getElementById('badgeProjects');
  const badgeDeployments = document.getElementById('badgeDeployments');
  const badgeWebhooks = document.getElementById('badgeWebhooks');

  // Projects View
  const projectsGrid = document.getElementById('projectsGrid');
  const emptyProjects = document.getElementById('emptyProjects');

  // Deployments View
  const deploymentsTbody = document.getElementById('deploymentsTbody');
  const emptyDeployments = document.getElementById('emptyDeployments');

  // Webhooks View
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

  const toast = document.getElementById('toast');

  // --- Initialize ---
  init();

  function init() {
    setupEventListeners();
    fetchAllData();
    startAutoRefresh();
  }

  function setupEventListeners() {
    // Tab Switching
    tabBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const targetTab = btn.getAttribute('data-tab');
        tabBtns.forEach(b => b.classList.remove('active'));
        tabContents.forEach(c => c.classList.remove('active'));

        btn.classList.add('active');
        document.getElementById(targetTab).classList.add('active');
      });
    });

    // Header buttons
    if (btnNewProject) {
      btnNewProject.addEventListener('click', () => openProjectModal());
    }

    if (btnRefresh) {
      btnRefresh.addEventListener('click', () => fetchAllData());
    }

    if (btnClear) {
      btnClear.addEventListener('click', clearWebhooks);
    }

    // Auto-refresh toggle
    autoRefreshToggle.addEventListener('change', (e) => {
      isAutoRefreshOn = e.target.checked;
      if (isAutoRefreshOn) {
        liveDot.classList.remove('paused');
        liveStatus.textContent = 'Auto-refresh On';
        startAutoRefresh();
      } else {
        liveDot.classList.add('paused');
        liveStatus.textContent = 'Paused';
        stopAutoRefresh();
      }
    });

    // Search and Filter
    if (searchInput) {
      searchInput.addEventListener('input', renderWebhooks);
    }
    if (eventFilter) {
      eventFilter.addEventListener('change', renderWebhooks);
    }

    // Project Modal
    if (btnCloseProjectModal) btnCloseProjectModal.addEventListener('click', closeProjectModal);
    if (btnCancelProjectModal) btnCancelProjectModal.addEventListener('click', closeProjectModal);
    if (projectForm) projectForm.addEventListener('submit', saveProject);

    // Terminal Modal
    if (btnCloseTerminalModal) btnCloseTerminalModal.addEventListener('click', closeTerminalModal);
    if (btnCopyLog) {
      btnCopyLog.addEventListener('click', () => {
        navigator.clipboard.writeText(terminalContent.textContent);
        showToast('Log copied to clipboard!');
      });
    }

    // Close modals on overlay click
    window.addEventListener('click', (e) => {
      if (e.target === projectModal) closeProjectModal();
      if (e.target === terminalModal) closeTerminalModal();
    });
  }

  // --- Auto-Refresh Interval ---
  function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshInterval = setInterval(() => {
      if (isAutoRefreshOn) {
        fetchAllData(true);
      }
    }, 4000);
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) {
      clearInterval(autoRefreshInterval);
      autoRefreshInterval = null;
    }
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

  // --- Rendering Projects ---
  function renderProjects() {
    badgeProjects.textContent = projects.length;
    if (projects.length === 0) {
      projectsGrid.innerHTML = '';
      emptyProjects.classList.remove('hidden');
      return;
    }

    emptyProjects.classList.add('hidden');
    projectsGrid.innerHTML = projects.map(p => `
      <div class="project-card">
        <div class="project-card-header">
          <div class="project-name-wrap">
            <span class="project-name">${escapeHTML(p.name)}</span>
            <span class="project-repo">${escapeHTML(p.repository)}</span>
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
            <span class="path-code">${escapeHTML(p.projectPath)}</span>
          </div>

          <div class="script-box">${escapeHTML(p.script || '# No script provided')}</div>
        </div>

        <div class="project-card-footer">
          <div style="display:flex; gap:0.4rem;">
            <button class="btn btn-xs btn-secondary" onclick="editProject('${p.id}')">Edit</button>
            <button class="btn btn-xs btn-danger-outline" onclick="deleteProject('${p.id}')">Delete</button>
          </div>
          <button class="btn btn-xs btn-success-outline" onclick="triggerDeploy('${p.id}')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="5 3 19 12 5 21 5 3"></polygon>
            </svg>
            Deploy Now
          </button>
        </div>
      </div>
    `).join('');
  }

  // --- Rendering Deployments ---
  function renderDeployments() {
    badgeDeployments.textContent = deployments.length;
    if (deployments.length === 0) {
      deploymentsTbody.innerHTML = '';
      emptyDeployments.classList.remove('hidden');
      return;
    }

    emptyDeployments.classList.add('hidden');
    deploymentsTbody.innerHTML = deployments.map(d => {
      const statusClass = d.status === 'SUCCESS' ? 'badge-status-success' : (d.status === 'FAILED' ? 'badge-status-failed' : 'badge-status-running');
      return `
        <tr>
          <td><span class="badge ${statusClass}">${d.status}</span></td>
          <td style="font-weight:700; color:white;">${escapeHTML(d.projectName || d.projectId)}</td>
          <td><span style="font-family:var(--font-mono); color:var(--accent-indigo);">${escapeHTML(d.repository)}</span></td>
          <td><span class="branch-badge">${escapeHTML(d.branch)}</span></td>
          <td><span class="sender-pill">${escapeHTML(d.triggeredBy || 'Manual')}</span></td>
          <td><span style="font-family:var(--font-mono);">${formatDuration(d.durationMs)}</span></td>
          <td><span class="time-stamp">${formatTime(d.startedAt)}</span></td>
          <td>
            <button class="btn btn-xs btn-secondary" onclick="openTerminalModal('${d.id}')">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="4 17 10 11 16 17 20 13"></polyline>
                <line x1="4" y1="7" x2="20" y2="7"></line>
              </svg>
              View Log
            </button>
          </td>
        </tr>
      `;
    }).join('');
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

    badgeWebhooks.textContent = filtered.length;

    if (filtered.length === 0) {
      payloadTbody.innerHTML = '';
      emptyState.classList.remove('hidden');
      return;
    }

    emptyState.classList.add('hidden');
    payloadTbody.innerHTML = filtered.map(ev => `
      <tr>
        <td>
          <span class="delivery-id" onclick="copyText('${escapeHTML(ev.deliveryID)}')">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            ${escapeHTML(ev.deliveryID)}
          </span>
        </td>
        <td><span class="badge badge-event-${escapeHTML(ev.eventType)}">${escapeHTML(ev.eventType)}</span></td>
        <td><span class="repo-name">${escapeHTML(ev.repositoryName || '-')}</span></td>
        <td>${ev.action ? `<span class="badge badge-action badge-action-${escapeHTML(ev.action)}">${escapeHTML(ev.action)}</span>` : '<span class="null-dash">-</span>'}</td>
        <td>${ev.prNumber ? `<span class="pr-number">#${ev.prNumber}</span>` : '<span class="null-dash">-</span>'}</td>
        <td>${ev.prTitle ? `<div class="pr-title" title="${escapeHTML(ev.prTitle)}">${escapeHTML(ev.prTitle)}</div>` : '<span class="null-dash">-</span>'}</td>
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

  // --- Update Stats ---
  function updateStats() {
    if (statProjects) statProjects.textContent = projects.filter(p => p.enabled).length;
    if (statDeployments) statDeployments.textContent = deployments.length;
    if (statSuccess) statSuccess.textContent = deployments.filter(d => d.status === 'SUCCESS').length;
    if (statFailed) statFailed.textContent = deployments.filter(d => d.status === 'FAILED').length;
  }

  // --- Project Modal Actions ---
  window.editProject = function(id) {
    const proj = projects.find(p => p.id === id);
    if (proj) openProjectModal(proj);
  };

  function openProjectModal(proj = null) {
    document.getElementById('projectId').value = proj ? proj.id : '';
    document.getElementById('projectName').value = proj ? proj.name : '';
    document.getElementById('projectRepo').value = proj ? proj.repository : '';
    document.getElementById('projectBranch').value = proj ? proj.branch : 'main';
    document.getElementById('projectPath').value = proj ? proj.projectPath : '';
    document.getElementById('projectScript').value = proj ? proj.script : 'cd /my/project\ngit pull origin main\ndocker compose down\ndocker compose up --build -d';
    document.getElementById('projectEnabled').checked = proj ? proj.enabled : true;

    modalProjectTitle.textContent = proj ? 'Edit Deployment Project' : 'Add Deployment Project';
    projectModal.classList.remove('hidden');
  }

  function closeProjectModal() {
    projectModal.classList.add('hidden');
    projectForm.reset();
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

    try {
      const url = id ? `/api/projects/${id}` : '/api/projects';
      const method = id ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });

      if (!res.ok) throw new Error('Failed to save project');
      showToast(id ? 'Project updated successfully!' : 'Project created successfully!');
      closeProjectModal();
      fetchProjects();
    } catch (err) {
      showToast(err.message, true);
    }
  }

  window.deleteProject = async function(id) {
    if (!confirm('Are you sure you want to delete this project deployment config?')) return;
    try {
      const res = await fetch(`/api/projects/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete project');
      showToast('Project deleted');
      fetchProjects();
    } catch (err) {
      showToast(err.message, true);
    }
  };

  // --- Manual Deploy Trigger ---
  window.triggerDeploy = async function(id) {
    try {
      showToast('Triggering deployment...');
      const res = await fetch(`/api/projects/${id}/deploy`, { method: 'POST' });
      if (!res.ok) throw new Error('Failed to trigger deployment');
      const data = await res.json();
      showToast('Deployment launched!');
      
      // Switch to Deployments tab
      document.querySelector('[data-tab="deploymentsTab"]').click();
      fetchDeployments();

      if (data.deployment && data.deployment.id) {
        setTimeout(() => openTerminalModal(data.deployment.id), 500);
      }
    } catch (err) {
      showToast(err.message, true);
    }
  };

  // --- Terminal Modal ---
  window.openTerminalModal = async function(depId) {
    terminalTitle.textContent = `Deployment Log: ${depId}`;
    terminalContent.textContent = 'Loading execution log...';
    terminalModal.classList.remove('hidden');

    try {
      const res = await fetch(`/api/deployments/${depId}`);
      if (!res.ok) throw new Error('Log not found');
      const data = await res.json();
      const dep = data.deployment;
      terminalTitle.textContent = `Deployment Console — ${dep.projectName} (${dep.repository}:${dep.branch})`;
      terminalContent.textContent = dep.log || 'No log output recorded.';
    } catch (err) {
      terminalContent.textContent = `Failed to load log: ${err.message}`;
    }
  };

  function closeTerminalModal() {
    terminalModal.classList.add('hidden');
  }

  // --- Clear Webhooks ---
  async function clearWebhooks() {
    if (!confirm('Clear all webhook history?')) return;
    try {
      const res = await fetch('/api/webhooks/clear', { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to clear webhooks');
      showToast('Webhook history cleared');
      fetchWebhooks();
    } catch (err) {
      showToast(err.message, true);
    }
  }

  // --- Helpers ---
  function showToast(msg, isError = false) {
    toast.textContent = msg;
    toast.style.borderColor = isError ? 'var(--accent-rose)' : 'var(--accent-indigo)';
    toast.classList.remove('hidden');
    setTimeout(() => {
      toast.classList.add('hidden');
    }, 3000);
  }

  window.copyText = function(text) {
    navigator.clipboard.writeText(text);
    showToast('Copied to clipboard!');
  };

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

  function formatDuration(ms) {
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
