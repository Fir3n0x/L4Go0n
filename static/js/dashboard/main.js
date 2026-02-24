// WebSocket connection
const ws = new WebSocket("ws://localhost:8080/ws");

// Load modules
document.addEventListener('DOMContentLoaded', () => {
  // Gloabl init
  loadLogs();
  loadCommands();
  loadConnections();
  loadReports();
  document.getElementById('command-filter').addEventListener('input', loadCommands);
  document.getElementById('connection-filter').addEventListener('input', loadConnections);
  document.getElementById('report-filter').addEventListener('input', loadReports);
  document.getElementById('agent-filter').addEventListener('input', loadAgents)
});

// Handle WebSocket messages
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === "report_update") {
    const id = data.id ? data.id : "";

    // if already exist, remove
    const idx = updatedOrder.indexOf(id);
    if (idx !== -1) updatedOrder.splice(idx, 1);

    updatedReports.add(data.id);


    // Handle terminal command
    updateTerminalInterface(id);
    
    loadReports();
    loadCommands();
  } else if (data.type === "command_update") {
    loadCommands();
  } else if (data.type === "connection_update") {
    loadConnections();
    loadCommands();
    loadAgents();
    loadGraph();
    updateWatchedConnections();
    // if current terminal is open, close it
  } else if (data.type === "connection_update_down") {
    loadConnections();
    loadCommands();
    loadAgents();
    loadGraph();
    updateWatchedConnections();
    closeTerminal();
  }
  loadLogs();
};

// display terminal output for the given agent ID
function updateTerminalInterface(id) {

  const sendBtn = document.querySelector('#terminal-panel button');
  const spinner = document.getElementById('terminal-spinner');

  // If an order is pending for this agent
  if (pendingCommandsByAgent.has(id)) {
    const queue = pendingCommandsByAgent.get(id);
    if (queue.length > 0) {
      const currentCommand = queue[0]; // FIFO

      const agentN = agentName.get(id);
      const agentJsonName = `${agentN}-${id}`;
      const output = document.getElementById('terminal-output');

      // Retrieve the command output from the server
      fetch(`/api/report?id=${encodeURIComponent(agentJsonName)}`)
        .then(res => res.json())
        .then(report => {
          const entry = report.findLast(e => e.command === currentCommand);
          if (entry) {
            const result = entry.output ?? `Unknown command: ${currentCommand}`;

            const commandLine = `<div class="text-green-400 font-mono">${agentN}:~\$ ${currentCommand}</div>`;
            const outputLine = `<pre class="text-white whitespace-pre-wrap font-mono mb-4">${result}</pre>`;

            output.innerHTML += commandLine + outputLine;
            output.scrollTop = output.scrollHeight;

            // Delete the processed order
            queue.shift();
          }
        })
        .finally(() => {
          isCommandRunning = false;
          sendBtn.disabled = false;
          sendBtn.classList.remove('opacity-50', 'cursor-not-allowed');
          spinner.classList.add('hidden');
        })
        .catch(err => {
          console.error("Error with report fetch :", err);
        });
    }
  }
}

// Handle tab switching
function showTab(tabId) {
  // Hide all tabs
  document.querySelectorAll('.tab').forEach(tab => {
    tab.classList.add('hidden');
    tab.classList.remove('active');
  });

  // Display the active tab
  const activeTab = document.getElementById(`${tabId}-tab`);
  if (activeTab) {
    activeTab.classList.remove('hidden');
    activeTab.classList.add('active');
  }

  // Reset button styles
  document.querySelectorAll('.tab-button').forEach(btn => {
    btn.classList.remove('active-tab');
  });

  // Apply the style to the active button
  const activeBtn = document.getElementById(`tab-${tabId}`);
  if (activeBtn) {
    activeBtn.classList.add('active-tab');
  }

  if (tabId === 'agents') loadAgents();
  else if (tabId === 'graph') {
    loadGraph();
  } else if (tabId === 'preset') {
    loadPresets();
  }
  loadConnections();
  loadCommands();
}

// logout function
function logout() {
  fetch('/logout', {
    method: 'GET',
    credentials: 'include' // include cookie
  })
  .then(() => {
    const toast = document.getElementById('message-toast');
    toast.classList.remove('hidden', 'opacity-0')
    toast.classList.add('opacity-100')

    setTimeout(() => {
      window.location.href = '/'; // redirect to login page
    }, 1000)

    setTimeout(() => {
      toast.classList.add('opacity-0')
      toast.addEventListener('transitionend', () => toast.classList.add('hidden'), { once: true })
    }, 2000)
  })
}

// delete a specific command for a given client ID
function deleteCommand(id, cmdText, index) {
  if (!confirm(`Delete command #${index} : "${cmdText}" for ${id} ?`)) return;
  fetch(`/api/del-command?id=${encodeURIComponent(id)}&cmd=${encodeURIComponent(cmdText)}&index=${index}`, {
    method: 'POST'
  })
  .then(res => res.text())
  .then(output => {
    // alert(`Deleted command: \n${cmdText} (#${index})\n\nResuslt:\n${output}`);

    loadCommands(); // Reload commands after deletion
    refreshModal(id);
    toggleCommandsModal(id);
  });
}

// delete a specific report execution command for a given filename and timestamp
function deleteExecutionCommand(filename, IDtimestamp) {
  if (!confirm(`Delete report execution command executed at ${IDtimestamp} ?`)) return;
  fetch(`/api/del-command-execution-report?IDtimestamp=${encodeURIComponent(IDtimestamp)}&filename=${encodeURIComponent(filename)}`, {
    method: 'POST'
  })
  .then(res => res.text())
  .then(() => {
    
    loadReports();

    const reportID = filename.replace(/\.json$/, "");
    
    fetch(`/api/report?id=${encodeURIComponent(reportID)}`)
      .then(res => res.json())
      .then(data => {
        toggleReportModal(filename, data)});
  })
}

// Flush all agents for all IDs
function flushAgents() {
  if (!confirm(`Flush all agents for all IDs?`)) return

  fetch(`/api/flush-agents`, {
    method: 'POST'
  })
  .then(res => res.text())
  .then(output => {
    // alert(`All agents flushed globally.\n\nResult:\n${output}`);
    watched.clear();
    agentName.clear();
    updateWatchedConnections();
  })
  .catch(err => {
    alert(`Flush failed: ${err}`);
    console.error(err);
  });
}

// Flush all commands for all IDs
function flushCommands() {
  if (!confirm(`Flush all commands for all IDs?`)) return

  fetch(`/api/flush-commands`, {
    method: 'POST'
  })
  .then(res => res.text())
  .then(output => {
    // alert(`All commands flushed globally.\n\nResult:\n${output}`);
    loadCommands();       // reload main list
  })
  .catch(err => {
    alert(`Flush failed: ${err}`);
    console.error(err);
  });
}

// Flush all reports for all IDs
function flushReports() {
  if (!confirm(`Flush all reports for all IDs?`)) return

  fetch(`/api/flush-reports`, {
    method: 'POST'
  })
  .then(res => res.text())
  .then(output => {
    // alert(`All reports flushed globally.\n\nResult:\n${output}`);
    loadReports();       // reload main list
  })
  .catch(err => {
    alert(`Flush failed: ${err}`);
    console.error(err);
  });

  loadReports();
}

// delete a specific connection by its ID
function delConnection(id) {
  if (!confirm(`Delete connection #${id} ? `)) return;
  fetch(`/api/del-connection?id=${encodeURIComponent(id)}`, {
    method: 'POST'
  })

  // Remove ID watched list
  agentName.delete(id);
  watched.delete(id);

  if (currentAgentID === id) {
    isCommandRunning = false;
    closeTerminal();
  }

  // alert(`Deleted connection: \n${id}`);
  loadConnections();
  loadCommands();
}

// shut down a specific connection by its ID
function shutDownConnection(id) {
  if (!confirm(`Shut down connection #${id} ? `)) return;
  fetch(`/api/shut-down-connection?id=${encodeURIComponent(id)}`, {
    method: 'POST'
  })

  const sendBtn = document.querySelector('#terminal-panel button');
  const spinner = document.getElementById('terminal-spinner');

  // Remove ID watched list
  agentName.delete(id);
  watched.delete(id);

  if (currentAgentID === id) {
    isCommandRunning = false;
    sendBtn.disabled = false;
    sendBtn.classList.remove('opacity-50', 'cursor-not-allowed');
    spinner.classList.add('hidden');
    closeTerminal();
  }


  // alert(`Deleted connection: \n${id}`);
  loadConnections();
  loadCommands();
}

// update the list of watched connections
function updateWatchedConnections() {
  fetch('/api/connections')
    .then(res => res.json())
    .then(data => {
      const container = document.getElementById('watched-list');
      container.innerHTML = '';

      for (const id of watched) {
        const info = data[id];
        const name = info ? info.name : 'Unknown';

        const tag = document.createElement('div');
        tag.className = 'font-mono text-xs bg-gray-800 p-1 rounded mb-1';
        tag.textContent = `$> ${name}-${id} is ready`;

        container.appendChild(tag);
      }
    });
}

// send command to all watched connections
function sendCommand() {
  const cmd = document.getElementById('in-command').value

  if (!watched.size) {
    alert("No agent selected...")
    return
  }

  if (cmd.trim().length === 0) {
    alert("No command provided...")
    return
  }


  for (const id of watched) {
    // if(!confirm(`Send command '${cmd}' to #${id} ? `)) continue;
    fetch(`/api/send-command?id=${encodeURIComponent(id)}&cmd=${encodeURIComponent(cmd)}`, {
      method: 'POST'
    });
    loadCommands();
  }

  document.getElementById('in-command').value = '';
}

document.getElementById('in-command').addEventListener('keydown', function (e) {
  if (e.key === 'Enter') {
    sendCommand();
  }
});

// select all connections (in active tab)
function selectAllConnection() {
  document
    .querySelectorAll('.connection-card')
    .forEach(card => {
      const id = card
        .querySelector('h4[data-id]')
        ?.dataset
        .id;
      if (id) {
        watched.add(id);
      }
    });

  // regenerate list to apply style changes
  loadConnections();
  updateWatchedConnections();
}

// unselect all connections (in active tab)
function unselectAllConnection() {
  document
    .querySelectorAll('.connection-card')
    .forEach(card => {
      const id = card
        .querySelector('h4[data-id]')
        ?.dataset
        .id;
      if (id) {
        watched.delete(id);
      }
    });

  loadConnections();
  updateWatchedConnections();
}

// Allow clickable button over clickable background
function handleDelete(event, id) {
  event.stopPropagation();
  delConnection(id);
}

// Toggle sidebar visibility
function toggleSidebar() {
  const sidebar = document.getElementById('sidebar');
  sidebar.classList.toggle('collapsed');

  // Adjusts hand width accordingly
  const main = document.querySelector('main');
  main.classList.toggle('w-4/5');
  main.classList.toggle('w-full');
}