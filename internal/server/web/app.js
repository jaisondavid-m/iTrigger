document.addEventListener('DOMContentLoaded', () => {
  let eventsData = [];
  let autoRefreshTimer = null;

  // DOM elements
  const payloadTbody = document.getElementById('payloadTbody');
  const emptyState = document.getElementById('emptyState');
  const searchInput = document.getElementById('searchInput');
  const eventFilter = document.getElementById('eventFilter');
  const autoRefreshToggle = document.getElementById('autoRefreshToggle');
  const liveDot = document.getElementById('liveDot');
  const liveStatus = document.getElementById('liveStatus');

  const statTotal = document.getElementById('statTotal');
  const statPR = document.getElementById('statPR');
  const statPush = document.getElementById('statPush');
  const statRepos = document.getElementById('statRepos');

  const btnRefresh = document.getElementById('btnRefresh');
  const btnClear = document.getElementById('btnClear');

  // Fetch events from API
  async function fetchEvents() {
    try {
      const response = await fetch('/api/webhooks');
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const data = await response.json();
      eventsData = data.events || [];
      render();
    } catch (err) {
      console.error('Failed to fetch events:', err);
    }
  }

  // Render UI
  function render() {
    const query = searchInput.value.toLowerCase().trim();
    const filterType = eventFilter.value;

    const filtered = eventsData.filter(evt => {
      const matchesFilter = (filterType === 'ALL') || (evt.eventType === filterType);
      
      const searchableStr = [
        evt.deliveryID,
        evt.eventType,
        evt.repositoryName,
        evt.action,
        evt.prNumber ? `#${evt.prNumber}` : '',
        evt.prTitle,
        evt.sender
      ].join(' ').toLowerCase();

      const matchesQuery = !query || searchableStr.includes(query);
      return matchesFilter && matchesQuery;
    });

    // Update Stats
    statTotal.textContent = eventsData.length;
    statPR.textContent = eventsData.filter(e => e.eventType === 'pull_request').length;
    statPush.textContent = eventsData.filter(e => e.eventType === 'push').length;
    
    const uniqueRepos = new Set(eventsData.map(e => e.repositoryName).filter(Boolean));
    statRepos.textContent = uniqueRepos.size;

    // Render Table
    payloadTbody.innerHTML = '';

    if (filtered.length === 0) {
      emptyState.classList.remove('hidden');
    } else {
      emptyState.classList.add('hidden');
      filtered.forEach(evt => {
        const row = document.createElement('tr');
        row.innerHTML = `
          <td>
            <span class="delivery-id" onclick="copyToClipboard('${escapeHtml(evt.deliveryID)}')" title="Click to copy Delivery ID">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
              </svg>
              ${escapeHtml(evt.deliveryID || '-')}
            </span>
          </td>
          <td>
            <span class="badge ${getEventBadgeClass(evt.eventType)}">
              ${escapeHtml(evt.eventType || '-')}
            </span>
          </td>
          <td>
            <span class="repo-name">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>
              </svg>
              ${escapeHtml(evt.repositoryName || '-')}
            </span>
          </td>
          <td>
            ${evt.action ? `<span class="badge badge-action ${getActionBadgeClass(evt.action)}">${escapeHtml(evt.action)}</span>` : '<span class="null-dash">-</span>'}
          </td>
          <td>
            ${evt.prNumber ? `<span class="pr-number">#${evt.prNumber}</span>` : '<span class="null-dash">-</span>'}
          </td>
          <td>
            ${evt.prTitle ? `<span class="pr-title" title="${escapeHtml(evt.prTitle)}">${escapeHtml(evt.prTitle)}</span>` : '<span class="null-dash">-</span>'}
          </td>
          <td>
            <span class="sender-pill">
              <span class="sender-avatar">${(evt.sender || 'U').charAt(0).toUpperCase()}</span>
              @${escapeHtml(evt.sender || '-')}
            </span>
          </td>
          <td>
            <span class="time-stamp">${formatTime(evt.receivedAt)}</span>
          </td>
        `;
        payloadTbody.appendChild(row);
      });
    }
  }

  // Event Badge Helpers
  function getEventBadgeClass(eventType) {
    if (eventType === 'pull_request') return 'badge-event-pull_request';
    if (eventType === 'push') return 'badge-event-push';
    if (eventType === 'ping') return 'badge-event-ping';
    return 'badge-event-default';
  }

  function getActionBadgeClass(action) {
    if (action === 'opened') return 'badge-action-opened';
    if (action === 'closed') return 'badge-action-closed';
    if (action === 'synchronize') return 'badge-action-synchronize';
    return '';
  }

  // Format timestamp
  function formatTime(isoStr) {
    if (!isoStr) return '-';
    try {
      const d = new Date(isoStr);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch {
      return isoStr;
    }
  }

  // Escape HTML helper
  function escapeHtml(str) {
    if (typeof str !== 'string') return str;
    return str.replace(/[&<>"']/g, m => ({
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#039;'
    }[m]));
  }

  // Copy helper
  window.copyToClipboard = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      showToast('Delivery ID copied to clipboard!');
    }).catch(err => {
      console.error('Failed to copy', err);
    });
  };

  // Toast Helper
  function showToast(msg) {
    const toast = document.getElementById('toast');
    toast.textContent = msg;
    toast.classList.remove('hidden');
    setTimeout(() => {
      toast.classList.add('hidden');
    }, 2500);
  }

  // Auto-refresh control
  function setupAutoRefresh() {
    if (autoRefreshToggle.checked) {
      liveDot.classList.remove('paused');
      liveStatus.textContent = 'Auto-refresh On';
      if (!autoRefreshTimer) {
        autoRefreshTimer = setInterval(fetchEvents, 3000);
      }
    } else {
      liveDot.classList.add('paused');
      liveStatus.textContent = 'Auto-refresh Off';
      if (autoRefreshTimer) {
        clearInterval(autoRefreshTimer);
        autoRefreshTimer = null;
      }
    }
  }

  // Event Listeners
  autoRefreshToggle.addEventListener('change', setupAutoRefresh);

  btnRefresh.addEventListener('click', () => {
    fetchEvents();
    showToast('Payloads refreshed');
  });

  btnClear.addEventListener('click', async () => {
    if (confirm('Are you sure you want to clear all webhook history?')) {
      try {
        await fetch('/api/webhooks/clear', { method: 'DELETE' });
        eventsData = [];
        render();
        showToast('Webhook history cleared');
      } catch (err) {
        console.error('Failed to clear events:', err);
      }
    }
  });

  searchInput.addEventListener('input', render);
  eventFilter.addEventListener('change', render);

  // Initial load
  fetchEvents();
  setupAutoRefresh();
});
