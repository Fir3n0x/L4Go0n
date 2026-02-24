// Handle Logs modal
function toggleLogsModal() {
  const modal = document.getElementById('logs-modal');
  const logsMain = document.getElementById('logs');
  const logsContent = document.getElementById('logs-modal-output');

  if (modal.classList.contains('hidden')) {
    // Displaying the modal
    logsContent.textContent = logsMain.textContent.trimStart();  // synchro when opened
    modal.classList.remove('hidden');
    modal.classList.add('logs-modal');
  } else {
    // Closing the modal
    modal.classList.add('hidden');
    modal.classList.remove('logs-modal');
  }

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      modal.classList.add('hidden');
    }
  });
  loadLogs();
}

// Handle Commands Modal
function toggleCommandsModal(id, commands) {
  const modal    = document.getElementById('commands-modal');
  const titleId  = document.getElementById('commands-modal-id');
  const content  = document.getElementById('commands-modal-content');

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      modal.classList.add('hidden');
    }
  });

  // Inject the client ID in the title
  titleId.textContent = agentName.get(id) + "-" + id;

  // Empty previous content
  content.innerHTML = "";

  // Generate each command row draggable
  commands.forEach((cmd, i) => {
    const row = document.createElement('div');
    row.className = 'command-item';
    row.setAttribute('draggable', 'true');
    row.setAttribute('data-index', i);

    // data-cmd for glitch
    row.innerHTML = `
      <div class="w-full grid grid-cols-[1fr_auto] items-start bg-gray-950 p-3 rounded text-green-200 text-sm gap-2">
        <span data-cmd="${cmd}" class="w-full break-words whitespace-pre-wrap overflow-hidden">${cmd}</span>
        <button class="px-2 py-1 text-red-400 hover:text-red-200 hover:bg-red-900 rounded text-sm"
                onclick="deleteCommand('${id}','${cmd}', '${i}')">✕</button>
      </div>
    `;
    content.appendChild(row);
  });

  // Init drag & drop
  initDragAndDrop(content, id);

  // Display the modal
  modal.classList.remove('hidden');
}

// Refresh the commands modal for a given client ID
function refreshModal(id) {
  fetch('/api/commands')
    .then(res => res.json())
    .then(data => {
      if (data[id]) {
        toggleCommandsModal(id, data[id]);
      }
    });
}

// Delete a command for a given client and refresh the modal
function updateCommandIndices(container, clientId) {
  const items = container.querySelectorAll('.command-item');
  items.forEach((item, i) => {
    item.setAttribute('data-index', i);

    const cmd = item.querySelector('span').textContent.trim();
    const btn = item.querySelector('button');

    btn.setAttribute('onclick', `deleteCommand('${clientId}', '${cmd}', ${i})`);
  })
}

// Drag & Drop functionality
function initDragAndDrop(container, clientId) {
  let dragged = null;

  container.querySelectorAll('.command-item').forEach(item => {
    item.addEventListener('dragstart', () => {
      dragged = item;
      item.classList.add('dragging');
    });
    item.addEventListener('dragend', () => {
      dragged.classList.remove('dragging');
      dragged = null;
    });
    item.addEventListener('dragover', e => e.preventDefault());
    item.addEventListener('drop', e => {
      e.preventDefault();
      if (dragged && dragged !== item) {
        container.insertBefore(dragged, item);

        updateCommandIndices(container, clientId);

        // send new order to server
        const newOrder = Array.from(container.children)
          .map(el => el.querySelector('span').textContent.trim());

        // alert("New order: " + newOrder.join(", "));

        fetch('/api/update-commands', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({id: clientId, commands: newOrder})
        })
        .then(res => res.text())
        .then(() => {
          loadCommands();
          refreshModal(clientId);
        })
        .catch(console.error);
      }
    });
  });
}

// close commands modal
function closeCommandsModal() {
  const modal = document.getElementById('commands-modal');
  modal.classList.remove('commands-modal');
  modal.classList.add('hidden');
}

// close agent modal
function closeAgentModal() {
  const modal = document.getElementById('create-agent-modal');
  modal.classList.remove('create-agent-modal');
  modal.classList.add('hidden');
}



// Handle report Modal
function toggleReportModal(filename, data) {
  const modal = document.getElementById('report-modal');
  const title = document.getElementById('modal-title');
  const body = document.getElementById('modal-body');

  title.textContent = filename;
  body.innerHTML = "";

  const sortedData = data.sort((a,b) => new Date(b.timestamp) - new Date(a.timestamp));

  sortedData.forEach(entry => {
    const block = document.createElement('div');
    block.className = "mb-6 border-b border-gray-400 pb-4";

    block.innerHTML = `
      <div class="flex justify-between items-center mb-4">
        <div class="text-green-400 text-xs mb-1">${entry.timestamp}</div>
        <button class="px-2 py-1 text-red-400 hover:text-red-200 hover:bg-red-900 rounded text-sm"
                onclick="deleteExecutionCommand('${filename}', '${entry.timestamp}')">✕</button>
      </div>
      <div class="text-green-300 font-bold mb-2">Command: <span class="text-green-100">${entry.command}</span></div>
      <pre class="bg-gray-950 p-3 rounded text-green-200 text-xs overflow-auto whitespace-pre-wrap">${entry.output}</pre>
    `;

    body.appendChild(block);
  })

  modal.classList.remove('hidden');

  // close modal
  const closeBtn = document.querySelector('.close-btn');
  closeBtn.onclick = () => modal.classList.add('hidden');

  document.onkeydown = (e) => {
    if (e.key === 'Escape') modal.classList.add('hidden');
  };
}

