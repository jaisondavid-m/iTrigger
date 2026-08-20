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

  // Sidebar Layout Navigation
  const sidebarNav = document.getElementById('sidebarNav');
  const navLinks = document.querySelectorAll('.nav-link');
  const tabContents = document.querySelectorAll('.tab-content');
  const viewTitle = document.getElementById('viewTitle');

  // Mobile Sidebar Elements
  const btnMenu = document.getElementById('btnMenu');
  const appSidebar = document.getElementById('appSidebar');
  const sidebarOverlayBackdrop = document.getElementById('sidebarOverlayBackdrop');
  
  // Logout Sidebar Button
  const btnLogoutSidebar = document.getElementById('btnLogoutSidebar');

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

  // Authentication & Settings Elements
  const loginOverlay = document.getElementById('loginOverlay');
  const loginForm = document.getElementById('loginForm');
  const settingsForm = document.getElementById('settingsForm');
  const btnLogout = document.getElementById('btnLogout');
  const appContainer = document.getElementById('appContainer');

  // Modals & Script Mode Toggle
  const projectModal = document.getElementById('projectModal');
  const projectForm = document.getElementById('projectForm');
  const modalProjectTitle = document.getElementById('modalProjectTitle');
  const btnCloseProjectModal = document.getElementById('btnCloseProjectModal');
  const btnCancelProjectModal = document.getElementById('btnCancelProjectModal');
  const btnScriptModeFile = document.getElementById('btnScriptModeFile');
  const btnScriptModeCustom = document.getElementById('btnScriptModeCustom');
  const scriptTextareaGroup = document.getElementById('scriptTextareaGroup');
  const scriptFileNotice = document.getElementById('scriptFileNotice');
  const projectScript = document.getElementById('projectScript');
  const projectSecret = document.getElementById('projectSecret');
  const btnGenerateSecret = document.getElementById('btnGenerateSecret');

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

  // Directory Browser Modal Elements
  const directoryModal = document.getElementById('directoryModal');
  const btnBrowsePath = document.getElementById('btnBrowsePath');
  const btnCloseDirectoryModal = document.getElementById('btnCloseDirectoryModal');
  const btnCancelDirectoryModal = document.getElementById('btnCancelDirectoryModal');
  const btnSelectDirectory = document.getElementById('btnSelectDirectory');

  const btnDirUp = document.getElementById('btnDirUp');
  const btnDirShortcutRoot = document.getElementById('btnDirShortcutRoot');
  const btnDirShortcutHome = document.getElementById('btnDirShortcutHome');
  const btnDirShortcutCwd = document.getElementById('btnDirShortcutCwd');

  const dirPathInput = document.getElementById('dirPathInput');
  const btnDirGo = document.getElementById('btnDirGo');
  const dirBreadcrumbs = document.getElementById('dirBreadcrumbs');
  const dirStatusText = document.getElementById('dirStatusText');
  const dirTags = document.getElementById('dirTags');
  const dirExplorerList = document.getElementById('dirExplorerList');
  const dirSelectedPath = document.getElementById('dirSelectedPath');

  let currentBrowsePath = '';
  let parentBrowsePath = '';
  let selectedBrowsePath = '';

  const toast = document.getElementById('toast');

  // --- Initialize ---
  init();

  async function init() {
    setupEventListeners();
    const authenticated = await checkAuth();
    if (authenticated) {
      fetchAllData();
      startAutoRefresh();
      startLiveTicker();
    }
  }

  let currentUser = null;
  let currentUserIsAdmin = false;
  let currentUserCanCreateProject = false;
  let usersList = [];

  async function checkAuth() {
    try {
      const res = await fetch('/api/auth/status');
      if (!res.ok) throw new Error('Unauthenticated');
      const data = await res.json();
      if (data.authenticated) {
        currentUser = data.username;
        currentUserIsAdmin = data.isAdmin || false;
        currentUserCanCreateProject = data.canCreateProject || false;

        if (loginOverlay) loginOverlay.classList.add('hidden');
        if (appContainer) appContainer.classList.remove('hidden');
        
        // Update newUsername field in Settings form
        const newUsernameEl = document.getElementById('newUsername');
        if (newUsernameEl) newUsernameEl.value = currentUser;

        // Toggle UI elements based on permissions
        const navWebhooks = document.querySelector('button[data-tab="webhooksTab"]');
        if (navWebhooks) {
          if (currentUserIsAdmin) navWebhooks.classList.remove('hidden');
          else navWebhooks.classList.add('hidden');
        }
        
        const navUsers = document.getElementById('navUsers');
        if (navUsers) {
          if (currentUserIsAdmin) navUsers.classList.remove('hidden');
          else navUsers.classList.add('hidden');
        }
        
        const btnNewProject = document.getElementById('btnNewProject');
        if (btnNewProject) {
          if (currentUserIsAdmin || currentUserCanCreateProject) btnNewProject.classList.remove('hidden');
          else btnNewProject.classList.add('hidden');
        }

        const btnBrowsePath = document.getElementById('btnBrowsePath');
        if (btnBrowsePath) {
          if (currentUserIsAdmin) btnBrowsePath.classList.remove('hidden');
          else btnBrowsePath.classList.add('hidden');
        }
        
        return true;
      } else {
        throw new Error('Unauthenticated');
      }
    } catch (err) {
      currentUser = null;
      currentUserIsAdmin = false;
      currentUserCanCreateProject = false;
      if (loginOverlay) loginOverlay.classList.remove('hidden');
      if (appContainer) appContainer.classList.add('hidden');
      stopAutoRefresh();
      return false;
    }
  }

  async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;
    const loginBtn = document.getElementById('btnLoginSubmit');
    const origText = loginBtn ? loginBtn.innerHTML : '';
    if (loginBtn) {
      loginBtn.disabled = true;
      loginBtn.innerHTML = `
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spin">
          <circle cx="12" cy="12" r="10"></circle>
          <path d="M12 2a10 10 0 0 1 10 10"></path>
        </svg>
        Authenticating...
      `;
    }
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      if (!res.ok) {
        if (res.status === 401) throw new Error('Invalid username or password');
        throw new Error('Failed to authenticate');
      }
      showToast('Authentication successful!');
      if (loginForm) loginForm.reset();
      
      const authenticated = await checkAuth();
      if (authenticated) {
        fetchAllData();
        startAutoRefresh();
        startLiveTicker();
      }
    } catch (err) {
      showToast(err.message, true);
    } finally {
      if (loginBtn) {
        loginBtn.disabled = false;
        loginBtn.innerHTML = origText;
      }
    }
  }

  async function handleLogout() {
    if (!confirm('Are you sure you want to log out?')) return;
    try {
      const res = await fetch('/api/auth/logout', { method: 'POST' });
      if (!res.ok) throw new Error('Failed to log out');
      showToast('Logged out successfully');
      checkAuth();
    } catch (err) {
      showToast(err.message, true);
    }
  }

  async function handleSaveSettings(e) {
    e.preventDefault();
    const currentPassword = document.getElementById('currentPassword').value;
    const newUsername = document.getElementById('newUsername').value;
    const newPassword = document.getElementById('newPassword').value;
    const saveBtn = document.getElementById('btnSaveSettings');
    const origText = saveBtn ? saveBtn.innerHTML : '';
    if (saveBtn) {
      saveBtn.disabled = true;
      saveBtn.innerHTML = `
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="spin">
          <circle cx="12" cy="12" r="10"></circle>
          <path d="M12 2a10 10 0 0 1 10 10"></path>
        </svg>
        Saving Changes...
      `;
    }
    try {
      const res = await fetch('/api/auth/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ currentPassword, newUsername, newPassword })
      });
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Failed to update credentials');
      }
      showToast('Credentials updated successfully!');
      if (settingsForm) settingsForm.reset();
      
      await checkAuth();
    } catch (err) {
      showToast(err.message, true);
    } finally {
      if (saveBtn) {
        saveBtn.disabled = false;
        saveBtn.innerHTML = origText;
      }
    }
  }

  function setupEventListeners() {
    // 0. Script Mode Segmented Buttons
    if (btnScriptModeFile && btnScriptModeCustom) {
      btnScriptModeFile.addEventListener('click', () => setScriptMode('file'));
      btnScriptModeCustom.addEventListener('click', () => setScriptMode('custom'));
    }

    // 1. Sidebar Page Switching
    if (sidebarNav) {
      sidebarNav.addEventListener('click', (e) => {
        const link = e.target.closest('.nav-link');
        if (!link) return;
        const targetTab = link.getAttribute('data-tab');
        
        navLinks.forEach(b => b.classList.remove('active'));
        tabContents.forEach(c => c.classList.remove('active'));

        link.classList.add('active');
        const contentEl = document.getElementById(targetTab);
        if (contentEl) contentEl.classList.add('active');

        // Update header view title
        if (viewTitle) {
          const spanText = link.querySelector('span').textContent;
          if (spanText === 'Projects') {
            viewTitle.textContent = 'Projects Dashboard';
          } else if (spanText === 'Deployment Logs') {
            viewTitle.textContent = 'System Logs';
          } else if (spanText === 'Webhook Payloads') {
            viewTitle.textContent = 'Webhook Payloads';
          } else if (spanText === 'Settings') {
            viewTitle.textContent = 'Account Settings';
          } else if (spanText === 'Users') {
            viewTitle.textContent = 'User Management';
          } else {
            viewTitle.textContent = spanText;
          }
        }

        if (targetTab === 'usersTab') {
          fetchUsers();
        }

        // Close mobile sidebar on select
        if (appSidebar && appSidebar.classList.contains('open')) {
          appSidebar.classList.remove('open');
          if (sidebarOverlayBackdrop) sidebarOverlayBackdrop.classList.add('hidden');
        }
      });
    }

    // Mobile Hamburger Menu Toggles
    if (btnMenu) {
      btnMenu.addEventListener('click', () => {
        if (appSidebar) appSidebar.classList.toggle('open');
        if (sidebarOverlayBackdrop) sidebarOverlayBackdrop.classList.toggle('hidden');
      });
    }

    if (sidebarOverlayBackdrop) {
      sidebarOverlayBackdrop.addEventListener('click', () => {
        if (appSidebar) appSidebar.classList.remove('open');
        sidebarOverlayBackdrop.classList.add('hidden');
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
        }
      });
    }

    // 7. Modals Listeners
    if (btnCloseProjectModal) btnCloseProjectModal.addEventListener('click', closeProjectModal);
    if (btnCancelProjectModal) btnCancelProjectModal.addEventListener('click', closeProjectModal);
    if (projectForm) projectForm.addEventListener('submit', saveProject);

    if (loginForm) loginForm.addEventListener('submit', handleLogin);
    if (settingsForm) settingsForm.addEventListener('submit', handleSaveSettings);
    if (btnLogout) btnLogout.addEventListener('click', handleLogout);
    if (btnLogoutSidebar) btnLogoutSidebar.addEventListener('click', handleLogout);

    if (btnGenerateSecret) {
      console.log("[iTrigger] btnGenerateSecret found. Attaching click listener.");
      btnGenerateSecret.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        console.log("[iTrigger] Generate webhook secret button clicked.");
        try {
          let secret = '';
          if (window.crypto && window.crypto.getRandomValues) {
            const array = new Uint8Array(20);
            window.crypto.getRandomValues(array);
            secret = Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
            console.log("[iTrigger] Generated secret using crypto.getRandomValues");
          } else {
            // Fallback for non-secure contexts / older browsers
            for (let i = 0; i < 40; i++) {
              secret += Math.floor(Math.random() * 16).toString(16);
            }
            console.log("[iTrigger] Generated secret using Math.random fallback");
          }
          if (projectSecret) {
            projectSecret.value = secret;
            showToast('Generated secure random secret!');
          } else {
            console.warn("[iTrigger] projectSecret input element not found.");
          }
        } catch (err) {
          console.error('Failed to generate secret:', err);
          showToast('Failed to generate secret. Please enter one manually.', true);
        }
      });
    } else {
      console.warn("[iTrigger] btnGenerateSecret button not found in DOM.");
    }

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
          const proj = projects.find(p => p.id === currentFailedProjectId);
          closeFailureModal();
          if (proj) {
            triggerDeploy(proj.id);
          }
        }
      });
    }

    // 8. Directory Browser Modal Listeners
    if (btnBrowsePath) {
      btnBrowsePath.addEventListener('click', () => {
        const inputVal = document.getElementById('projectPath').value;
        openDirectoryModal(inputVal);
      });
    }

    if (btnCloseDirectoryModal) btnCloseDirectoryModal.addEventListener('click', closeDirectoryModal);
    if (btnCancelDirectoryModal) btnCancelDirectoryModal.addEventListener('click', closeDirectoryModal);

    if (btnSelectDirectory) {
      btnSelectDirectory.addEventListener('click', () => {
        if (selectedBrowsePath) {
          document.getElementById('projectPath').value = selectedBrowsePath;
          showToast(`Selected server path: ${selectedBrowsePath}`);
        }
        closeDirectoryModal();
      });
    }

    if (btnDirUp) {
      btnDirUp.addEventListener('click', () => {
        if (parentBrowsePath) {
          loadDirectory(parentBrowsePath);
        }
      });
    }

    if (btnDirShortcutRoot) {
      btnDirShortcutRoot.addEventListener('click', () => loadDirectory('/'));
    }
    if (btnDirShortcutHome) {
      btnDirShortcutHome.addEventListener('click', () => loadDirectory('~'));
    }
    if (btnDirShortcutCwd) {
      btnDirShortcutCwd.addEventListener('click', () => loadDirectory('.'));
    }

    if (btnDirGo && dirPathInput) {
      btnDirGo.addEventListener('click', () => {
        if (dirPathInput.value.trim()) {
          loadDirectory(dirPathInput.value.trim());
        }
      });
      dirPathInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          if (dirPathInput.value.trim()) {
            loadDirectory(dirPathInput.value.trim());
          }
        }
      });
    }

    if (dirBreadcrumbs) {
      dirBreadcrumbs.addEventListener('click', (e) => {
        const crumb = e.target.closest('[data-path]');
        if (crumb) {
          loadDirectory(crumb.dataset.path);
        }
      });
    }

    if (dirExplorerList) {
      dirExplorerList.addEventListener('click', (e) => {
        const openBtn = e.target.closest('[data-action="open-folder"]');
        if (openBtn) {
          e.stopPropagation();
          loadDirectory(openBtn.dataset.path);
          return;
        }

        const row = e.target.closest('.dir-row');
        if (!row) return;

        if (row.classList.contains('is-folder')) {
          const path = row.dataset.path;
          document.querySelectorAll('.dir-row').forEach(r => r.classList.remove('selected'));
          row.classList.add('selected');
          selectedBrowsePath = path;
          if (dirSelectedPath) dirSelectedPath.textContent = path;

          if (e.detail === 2) {
            loadDirectory(path);
          }
        }
      });
    }

    // Close modals on overlay click or Escape key
    window.addEventListener('click', (e) => {
      if (e.target === projectModal) closeProjectModal();
      if (e.target === terminalModal) closeTerminalModal();
      if (e.target === failureModal) closeFailureModal();
      if (e.target === userModal) closeUserModal();
    });

    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        closeProjectModal();
        closeTerminalModal();
        closeFailureModal();
        closeUserModal();
      }
    });

    // User modal buttons and form submit
    const btnNewUser = document.getElementById('btnNewUser');
    if (btnNewUser) {
      btnNewUser.addEventListener('click', () => openUserModal());
    }
    const btnCancelUserModal = document.getElementById('btnCancelUserModal');
    if (btnCancelUserModal) {
      btnCancelUserModal.addEventListener('click', closeUserModal);
    }
    const btnCloseUserModal = document.getElementById('btnCloseUserModal');
    if (btnCloseUserModal) {
      btnCloseUserModal.addEventListener('click', closeUserModal);
    }
    const userForm = document.getElementById('userForm');
    if (userForm) {
      userForm.addEventListener('submit', saveUser);
    }
    const usersTbody = document.getElementById('usersTbody');
    if (usersTbody) {
      usersTbody.addEventListener('click', (e) => {
        const btn = e.target.closest('button');
        if (!btn) return;
        const action = btn.getAttribute('data-user-action');
        const uName = btn.getAttribute('data-username');
        if (action === 'edit') {
          const u = usersList.find(usr => usr.username === uName);
          if (u) openUserModal(u);
        } else if (action === 'delete') {
          deleteUser(uName);
        }
      });
    }
  }

  function setScriptMode(mode) {
    if (mode === 'custom') {
      if (btnScriptModeCustom) btnScriptModeCustom.classList.add('active');
      if (btnScriptModeFile) btnScriptModeFile.classList.remove('active');
      if (scriptTextareaGroup) scriptTextareaGroup.classList.remove('hidden');
      if (scriptFileNotice) scriptFileNotice.classList.add('hidden');
    } else {
      if (btnScriptModeFile) btnScriptModeFile.classList.add('active');
      if (btnScriptModeCustom) btnScriptModeCustom.classList.remove('active');
      if (scriptTextareaGroup) scriptTextareaGroup.classList.add('hidden');
      if (scriptFileNotice) scriptFileNotice.classList.remove('hidden');
    }
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
    const promises = [
      fetchProjects(silent),
      fetchDeployments(silent)
    ];
    if (currentUserIsAdmin) {
      promises.push(fetchWebhooks(silent));
    }
    await Promise.all(promises);
  }

  async function fetchProjects(silent = false) {
    try {
      const res = await fetch('/api/projects');
      if (res.status === 401) {
        checkAuth();
        return;
      }
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
      if (res.status === 401) {
        checkAuth();
        return;
      }
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
      if (res.status === 401) {
        checkAuth();
        return;
      }
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
      projectsGrid.innerHTML = projects.map(p => {
        const hasCustomScript = p.script && p.script.trim().length > 0 && !p.script.startsWith('# Auto-detect');
        const isRead = p.userPermission === 'read';
        return `
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
              <div style="display: flex; gap: 0.4rem; align-items: center;">
                <span class="badge ${p.enabled ? 'badge-status-success' : 'badge-status-failed'}">
                  ${p.enabled ? 'Enabled' : 'Disabled'}
                </span>
                ${isRead ? `<span class="badge badge-status-failed" style="background:rgba(245,158,11,0.15); color:#fbbf24; border:1px solid rgba(245,158,11,0.3)">Read-Only</span>` : ''}
              </div>
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

              <div class="project-meta-row">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                  <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
                </svg>
                <span>Webhook Secret:</span>
                ${p.secret ? `
                  <span class="path-code" data-action="copy-path" data-path="${escapeHTML(p.secret)}" title="Click to copy secret">•••••••• (click to copy)</span>
                ` : `
                  <span style="color:var(--text-muted); font-style:italic;">None (uses global default)</span>
                `}
              </div>

              <div class="script-box" style="${hasCustomScript ? '' : 'color:var(--accent-cyan); font-style:italic;'}">
                ${hasCustomScript ? escapeHTML(p.script) : '⚡ Auto-discovering .itrigger script in repo root'}
              </div>
            </div>

            ${isRead ? '' : `
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
            `}
          </div>
        `;
      }).join('');
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
          <td><span style="font-family:var(--font-mono); color:var(--text-primary); font-weight:600;">${escapeHTML(ev.repositoryName || '-')}</span></td>
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
    if (projectSecret) projectSecret.value = proj ? (proj.secret || '') : '';
    document.getElementById('projectEnabled').checked = proj ? proj.enabled : true;

    const hasCustomScript = proj && proj.script && proj.script.trim().length > 0 && !proj.script.startsWith('# Auto-detect');
    if (hasCustomScript) {
      setScriptMode('custom');
      if (projectScript) projectScript.value = proj.script;
    } else {
      setScriptMode('file');
      if (projectScript) projectScript.value = '';
    }

    if (modalProjectTitle) modalProjectTitle.textContent = proj ? 'Edit Deployment Project' : 'Add Deployment Project';
    if (projectModal) projectModal.classList.remove('hidden');
  }

  function closeProjectModal() {
    if (projectModal) projectModal.classList.add('hidden');
    if (projectForm) projectForm.reset();
    setScriptMode('file');
  }

  async function saveProject(e) {
    e.preventDefault();
    const id = document.getElementById('projectId').value;
    const isCustomMode = btnScriptModeCustom && btnScriptModeCustom.classList.contains('active');
    const scriptValue = isCustomMode ? (projectScript ? projectScript.value : '') : '';

    const body = {
      name: document.getElementById('projectName').value,
      repository: document.getElementById('projectRepo').value,
      branch: document.getElementById('projectBranch').value,
      projectPath: document.getElementById('projectPath').value,
      script: scriptValue,
      secret: projectSecret ? projectSecret.value : '',
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

  // --- Server Directory Browser Modal Logic ---
  function openDirectoryModal(initialPath) {
    if (directoryModal) directoryModal.classList.remove('hidden');
    loadDirectory(initialPath || '');
  }

  function closeDirectoryModal() {
    if (directoryModal) directoryModal.classList.add('hidden');
  }

  async function loadDirectory(pathStr) {
    if (dirStatusText) dirStatusText.textContent = 'Listing server directory...';
    if (dirExplorerList) {
      dirExplorerList.innerHTML = `
        <div style="padding:2.5rem; text-align:center; color:var(--text-muted);">
          <div class="spin" style="display:inline-block; margin-bottom:0.5rem;">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21.5 2v6h-6M2.13 15.57a10 10 0 0 0 16.59 4.34M2.5 22v-6h6M21.87 8.43A10 10 0 0 0 5.28 4.09"/>
            </svg>
          </div>
          <div>Reading directory contents...</div>
        </div>
      `;
    }

    try {
      const url = '/api/fs/browse' + (pathStr ? `?path=${encodeURIComponent(pathStr)}` : '');
      const res = await fetch(url);
      if (res.status === 401) {
        checkAuth();
        return;
      }
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Failed to list directory');
      }
      const data = await res.json();
      if (data.status !== 'success') throw new Error('Invalid response from server');

      currentBrowsePath = data.currentPath || '';
      parentBrowsePath = data.parentPath || '';
      selectedBrowsePath = data.currentPath || '';

      if (dirPathInput) dirPathInput.value = currentBrowsePath;
      if (dirSelectedPath) dirSelectedPath.textContent = currentBrowsePath;

      renderBreadcrumbs(currentBrowsePath);
      renderDirTags(data.isRepo, data.hasTriggerScript);
      renderDirectoryList(data.folders || [], data.files || []);

      if (dirStatusText) {
        dirStatusText.textContent = `Path: ${currentBrowsePath} (${(data.folders || []).length} folders, ${(data.files || []).length} files)`;
      }
    } catch (err) {
      if (dirStatusText) dirStatusText.textContent = `Error: ${err.message}`;
      if (dirExplorerList) {
        dirExplorerList.innerHTML = `
          <div style="padding:2rem; text-align:center; color:var(--accent-rose);">
            <strong>Failed to access directory:</strong> ${escapeHTML(err.message)}
          </div>
        `;
      }
    }
  }

  function renderBreadcrumbs(fullPath) {
    if (!dirBreadcrumbs) return;
    if (!fullPath) {
      dirBreadcrumbs.innerHTML = '<span class="crumb-item" data-path="/">/</span>';
      return;
    }

    const isWin = fullPath.includes(':');
    const parts = fullPath.split(/[/\\]/).filter(Boolean);
    let html = '';
    let accPath = isWin ? '' : '/';

    if (!isWin) {
      html += `<span class="crumb-item" data-path="/">/</span>`;
    }

    parts.forEach((part, idx) => {
      if (isWin && idx === 0) {
        accPath = part;
        if (!accPath.endsWith('/')) accPath += '/';
      } else {
        if (!accPath.endsWith('/') && accPath !== '') accPath += '/';
        accPath += part;
      }

      if (idx > 0 || !isWin) {
        html += `<span class="crumb-sep">/</span>`;
      }
      html += `<span class="crumb-item" data-path="${escapeHTML(accPath)}">${escapeHTML(part)}</span>`;
    });

    dirBreadcrumbs.innerHTML = html;
  }

  function renderDirTags(isRepo, hasTriggerScript) {
    if (!dirTags) return;
    let html = '';
    if (isRepo) {
      html += `<span class="tag-badge git">Git Repository</span>`;
    }
    if (hasTriggerScript) {
      html += `<span class="tag-badge script">.itrigger Configured</span>`;
    }
    dirTags.innerHTML = html;
  }

  function renderDirectoryList(folders, files) {
    if (!dirExplorerList) return;

    if (folders.length === 0 && files.length === 0) {
      dirExplorerList.innerHTML = `
        <div style="padding:2.5rem; text-align:center; color:var(--text-muted);">
          <div style="font-size:1.5rem; margin-bottom:0.4rem;">📂</div>
          <div>Directory is empty</div>
        </div>
      `;
      return;
    }

    let rowsHTML = '';

    folders.forEach(f => {
      let badges = '';
      if (f.isRepo) badges += `<span class="badge-mini badge-mini-git">git</span>`;
      if (f.hasTriggerScript) badges += `<span class="badge-mini badge-mini-script">.itrigger</span>`;

      const modDate = f.modTime ? formatTime(f.modTime) : '-';

      rowsHTML += `
        <div class="dir-row is-folder" data-path="${escapeHTML(f.path)}">
          <div class="dir-row-name">
            <span class="dir-icon icon-folder-svg">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
              </svg>
            </span>
            <span>${escapeHTML(f.name)}</span>
            <div class="dir-badges-inline">${badges}</div>
          </div>
          <div class="dir-row-date">${modDate}</div>
          <div class="dir-row-action">
            <button type="button" class="btn-open-dir" data-action="open-folder" data-path="${escapeHTML(f.path)}" title="Open directory (cd ${escapeHTML(f.name)})">
              Open &rarr;
            </button>
          </div>
        </div>
      `;
    });

    files.forEach(f => {
      const modDate = f.modTime ? formatTime(f.modTime) : '-';

      rowsHTML += `
        <div class="dir-row is-file" data-path="${escapeHTML(f.path)}">
          <div class="dir-row-name">
            <span class="dir-icon icon-file-svg">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M13 2H6a2 2 0 0 1-2 2v16a2 2 0 0 1 2 2h12a2 2 0 0 1 2-2V9z"></path>
                <polyline points="13 2 13 9 20 9"></polyline>
              </svg>
            </span>
            <span>${escapeHTML(f.name)}</span>
          </div>
          <div class="dir-row-date">${modDate}</div>
          <div class="dir-row-action">
            <span style="font-size:0.75rem; color:var(--text-muted);">-</span>
          </div>
        </div>
      `;
    });

    dirExplorerList.innerHTML = rowsHTML;
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
    if (status === 'RUNNING') {
      if (startedAt) {
        const startMs = new Date(startedAt).getTime();
        if (!isNaN(startMs) && startMs > 0) {
          const nowMs = Date.now();
          const elapsedSec = Math.max(1, Math.floor((nowMs - startMs) / 1000));
          return `${elapsedSec}s`;
        }
      }
      return '1s';
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

  // --- User Management Client Handlers ---
  const userModal = document.getElementById('userModal');
  const userForm = document.getElementById('userForm');
  const userIsAdminCheckbox = document.getElementById('userIsAdmin');
  const projectPermissionsGroup = document.getElementById('projectPermissionsGroup');
  const userCanCreateProjectRow = document.getElementById('userCanCreateProjectRow');
  const modalProjectPermissionsTbody = document.getElementById('modalProjectPermissionsTbody');

  function openUserModal(user = null) {
    const titleEl = document.getElementById('modalUserTitle');
    const usernameInput = document.getElementById('userUsername');
    const passwordInput = document.getElementById('userPassword');
    const passwordReq = document.getElementById('userPasswordReq');
    const passwordHelp = document.getElementById('userPasswordHelp');

    if (user) {
      if (titleEl) titleEl.textContent = 'Edit User Account';
      if (usernameInput) {
        usernameInput.value = user.username;
        usernameInput.disabled = true;
      }
      if (passwordInput) {
        passwordInput.value = '';
        passwordInput.required = false;
      }
      if (passwordReq) passwordReq.style.display = 'none';
      if (passwordHelp) passwordHelp.style.display = 'inline';
      if (userIsAdminCheckbox) userIsAdminCheckbox.checked = user.isAdmin;
      const uCanCreate = document.getElementById('userCanCreateProject');
      if (uCanCreate) uCanCreate.checked = user.canCreateProject;
    } else {
      if (titleEl) titleEl.textContent = 'Add User Account';
      if (usernameInput) {
        usernameInput.value = '';
        usernameInput.disabled = false;
      }
      if (passwordInput) {
        passwordInput.value = '';
        passwordInput.required = true;
      }
      if (passwordReq) passwordReq.style.display = 'inline';
      if (passwordHelp) passwordHelp.style.display = 'none';
      if (userIsAdminCheckbox) userIsAdminCheckbox.checked = false;
      const uCanCreate = document.getElementById('userCanCreateProject');
      if (uCanCreate) uCanCreate.checked = true;
    }

    if (modalProjectPermissionsTbody) {
      modalProjectPermissionsTbody.innerHTML = projects.map(p => {
        let currentVal = 'none';
        if (user && user.permissions && user.permissions[p.id]) {
          currentVal = user.permissions[p.id];
        }
        return `
          <tr>
            <td>${escapeHTML(p.name)}</td>
            <td>
              <select class="form-input project-perm-select" data-project-id="${escapeHTML(p.id)}" style="padding: 0.35rem 0.5rem; font-size: 0.8rem; background: var(--bg-input);">
                <option value="none" ${currentVal === 'none' ? 'selected' : ''}>No Access</option>
                <option value="read" ${currentVal === 'read' ? 'selected' : ''}>Read-Only</option>
                <option value="write" ${currentVal === 'write' ? 'selected' : ''}>Write / Trigger</option>
              </select>
            </td>
          </tr>
        `;
      }).join('');
    }

    togglePermissionsGroupVisibility();

    if (userModal) userModal.classList.remove('hidden');
  }

  function togglePermissionsGroupVisibility() {
    const isAdmin = userIsAdminCheckbox && userIsAdminCheckbox.checked;
    if (isAdmin) {
      if (projectPermissionsGroup) projectPermissionsGroup.classList.add('hidden');
      if (userCanCreateProjectRow) userCanCreateProjectRow.classList.add('hidden');
    } else {
      if (projectPermissionsGroup) projectPermissionsGroup.classList.remove('hidden');
      if (userCanCreateProjectRow) userCanCreateProjectRow.classList.remove('hidden');
    }
  }

  if (userIsAdminCheckbox) {
    userIsAdminCheckbox.addEventListener('change', togglePermissionsGroupVisibility);
  }

  function closeUserModal() {
    if (userModal) userModal.classList.add('hidden');
    if (userForm) userForm.reset();
  }

  async function fetchUsers() {
    try {
      const res = await fetch('/api/users');
      if (!res.ok) throw new Error('Failed to fetch users');
      const data = await res.json();
      usersList = data;
      renderUsers();
    } catch (err) {
      showToast(err.message, true);
    }
  }

  function renderUsers() {
    const usersTbody = document.getElementById('usersTbody');
    if (!usersTbody) return;
    if (usersList.length === 0) {
      usersTbody.innerHTML = `<tr><td colspan="6" style="text-align:center; color:var(--text-secondary); padding: 2rem;">No user accounts found.</td></tr>`;
      return;
    }

    usersTbody.innerHTML = usersList.map(u => {
      const isSelf = u.username === currentUser;
      const isDefaultAdmin = u.username === 'itrigger';
      
      let projectsAccessText = '';
      if (u.isAdmin) {
        projectsAccessText = '<span style="color:var(--accent-indigo); font-weight:700;">Administrator (All Projects)</span>';
      } else {
        const list = [];
        for (const [projId, perm] of Object.entries(u.permissions)) {
          const p = projects.find(proj => proj.id === projId);
          if (p) {
            const permLabel = perm === 'write' ? 'Write' : 'Read';
            const color = perm === 'write' ? 'var(--accent-emerald)' : 'var(--text-secondary)';
            list.push(`<span style="color:${color}">${escapeHTML(p.name)} (${permLabel})</span>`);
          }
        }
        projectsAccessText = list.length > 0 ? list.join(', ') : '<span style="color:var(--text-muted); font-style:italic;">No project access</span>';
      }

      return `
        <tr>
          <td style="font-weight:600;">${escapeHTML(u.username)} ${isSelf ? ' <span style="font-weight:normal; font-size:0.75rem; color:var(--text-muted); font-style:italic;">(you)</span>' : ''}</td>
          <td>
            <span class="badge ${u.isAdmin ? 'badge-status-success' : 'badge-status-failed'}" style="${u.isAdmin ? 'background:rgba(99,102,241,0.15); color:var(--accent-indigo); border:1px solid rgba(99,102,241,0.3)' : ''}">
              ${u.isAdmin ? 'Admin' : 'User'}
            </span>
          </td>
          <td>
            <span class="badge ${u.canCreateProject && !u.isAdmin ? 'badge-status-success' : 'badge-status-failed'}">
              ${u.isAdmin || u.canCreateProject ? 'Yes' : 'No'}
            </span>
          </td>
          <td style="max-width:350px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title="${projectsAccessText.replace(/<[^>]*>/g, '')}">
            ${projectsAccessText}
          </td>
          <td class="time-stamp">${formatTime(u.createdAt)}</td>
          <td style="text-align: right;">
            <button type="button" class="btn btn-xs btn-secondary" data-user-action="edit" data-username="${escapeHTML(u.username)}">Edit</button>
            <button type="button" class="btn btn-xs btn-danger-outline" data-user-action="delete" data-username="${escapeHTML(u.username)}" ${isSelf || isDefaultAdmin ? 'disabled' : ''}>Delete</button>
          </td>
        </tr>
      `;
    }).join('');
  }

  async function saveUser(e) {
    e.preventDefault();
    const username = document.getElementById('userUsername').value;
    const isEdit = document.getElementById('userUsername').disabled;
    const password = document.getElementById('userPassword').value;
    const isAdmin = document.getElementById('userIsAdmin').checked;
    const canCreateProject = document.getElementById('userCanCreateProject').checked;

    const permissions = {};
    if (!isAdmin && modalProjectPermissionsTbody) {
      const selects = modalProjectPermissionsTbody.querySelectorAll('.project-perm-select');
      selects.forEach(sel => {
        const projId = sel.getAttribute('data-project-id');
        const val = sel.value;
        if (val !== 'none') {
          permissions[projId] = val;
        }
      });
    }

    const body = {
      isAdmin,
      canCreateProject,
      permissions
    };
    if (!isEdit) {
      body.username = username;
      body.password = password;
    } else if (password !== '') {
      body.password = password;
    }

    try {
      const url = isEdit ? `/api/users/${encodeURIComponent(username)}` : '/api/users';
      const method = isEdit ? 'PUT' : 'POST';
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });

      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Failed to save user account');
      }

      showToast(isEdit ? 'User updated successfully!' : 'User created successfully!');
      closeUserModal();
      fetchUsers();
    } catch (err) {
      showToast(err.message, true);
    }
  }

  async function deleteUser(username) {
    if (!confirm(`Are you sure you want to delete user "${username}"?`)) return;
    try {
      const res = await fetch(`/api/users/${encodeURIComponent(username)}`, {
        method: 'DELETE'
      });
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Failed to delete user account');
      }
      showToast('User deleted successfully!');
      fetchUsers();
    } catch (err) {
      showToast(err.message, true);
    }
  }
});
