// Command's list to add into the new preset
let currentCommands = []

// Create a new command template preset
function createPreset() {
  currentCommands = [];
  document.getElementById('preset-name').value = '';
  document.getElementById('command-input').value = '';
  document.getElementById('command-list').innerHTML = '';
  document.getElementById('new-preset-modal').classList.remove('hidden');

  makeModalDraggable('floating-content-window-preset', 'modal-header-create-preset');
}

// Close the preset creation modal
function closeNewCommandTemplateModal() {
    document.getElementById('new-preset-modal').classList.add('hidden');
}

// Handle closing modal with escape key
document.addEventListener('keydown', (e) => {
  const createPresetModal = document.getElementById('new-preset-modal');
  const isPresetCreationVisible = !createPresetModal.classList.contains('hidden');

  if (e.key === 'Escape' && isPresetCreationVisible) {
    createPresetModal.classList.add('hidden');
  }
})

// Handle adding command to preset with enter key
document.getElementById('command-input').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') {
    e.preventDefault();
    addCommandToPreset();
  }
});

// Add new command to the new preset
function addCommandToPreset() {
  const input = document.getElementById('command-input');
  const command = input.value.trim();
  if (!command) return;

  currentCommands.push(command);
  input.value = '';
  renderCommandList();
}

// Remove a command from the preset command in the current preset's creation
function removeCommand(index) {
  currentCommands.splice(index, 1);
  renderCommandList();
}

// Refresh command list in new preset command list
function renderCommandList() {
  const list = document.getElementById('command-list');
  list.innerHTML = '';

  currentCommands.forEach((cmd, index) => {
    const li = document.createElement('li');
    li.className = "bg-gray-700 px-3 py-2 rounded cursor-move";
    li.draggable = true;
    li.dataset.index = index;

    li.ondragstart = (e) => {
      e.dataTransfer.setData("text/plain", index);
    };

    li.ondragover = (e) => e.preventDefault();

    li.ondrop = (e) => {
      const from = parseInt(e.dataTransfer.getData("text/plain"));
      const to = parseInt(li.dataset.index);
      const moved = currentCommands.splice(from, 1)[0];
      currentCommands.splice(to, 0, moved);
      renderCommandList();
    };

    li.innerHTML = `
      <div class="relative w-full pr-8">
        <span class="block whitespace-pre-wrap break-words w-full">${cmd}</span>
        <button onclick="removeCommand(${index})" class="absolute top-0 right-0 text-red-400 hover:text-red-200">✖</button>
      </div>
    `;
    list.appendChild(li);
  });
}

// Save the new created preset
function savePreset() {
  const name = document.getElementById('preset-name').value.trim();
  if (!name || currentCommands.length === 0) {
    alert("Please enter a name and at least one command.");
    return;
  }

  const payload = {
    name: name,
    commands: currentCommands
  }

  fetch('/api/save-new-command-template-preset', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  })
    .then(response => {
      if (!response.ok) throw new Error("Failed to save preset");
      return response.json();
    })
    .then(data => {
      closeNewCommandTemplateModal();
      loadPresets();
    })
    .catch(err => {
      console.error("Error saving preset:", err);
      alert("Failed to save preset.");
    });
}

// Handle draggable option in new preset creation window
function makeModalDraggable(modalId, handleId) {
  const modal = document.getElementById(modalId);
  const handle = document.getElementById(handleId);
  let offsetX = 0, offsetY = 0, isDragging = false;

  handle.addEventListener('mousedown', (e) => {
    isDragging = true;
    offsetX = e.clientX - modal.offsetLeft;
    offsetY = e.clientY - modal.offsetTop;
    document.body.style.userSelect = 'none';
  });

  document.addEventListener('mousemove', (e) => {
    if (!isDragging) return;
    modal.style.left = `${e.clientX - offsetX}px`;
    modal.style.top = `${e.clientY - offsetY}px`;
  });

  document.addEventListener('mouseup', () => {
    isDragging = false;
    document.body.style.userSelect = '';
  });
}

// Handle key for new command template creation modal
document.addEventListener('keydown', (e) => {
  if (e.key !== 'Escape') return;

  const modals = [
    { id: 'new-preset-modal', close: closeNewCommandTemplateModal },
    { id: 'modify-preset-modal', close: closeModifyPresetModal }
  ];

  modals.forEach(({ id, close }) => {
    const modal = document.getElementById(id);
    if (modal && !modal.classList.contains('hidden')) {
      close();
    }
  });
});

// Delete a preset by its name
function deletePreset(name) {
  fetch(`/api/delete-preset/${encodeURIComponent(name)}`, {
    method: 'DELETE'
  })
  .then(res => {
    if (res.ok) {
      loadPresets(); // Refresh list
    } else {
      console.error('Failed to delete preset');
    }
  });
}

// Modify a preset already created
async function modifyPreset() {
  const selectedPreset = document.getElementById('selected-preset').textContent;
  if (selectedPreset === "NONE") {
    alert("Please select a preset to modify it.");
    return;
  }

  const res = await fetch('/api/command-template-presets');
  const presets = await res.json();

  const commands = presets[selectedPreset] || [];

  document.getElementById('modify-preset-name').value = selectedPreset;
  document.getElementById('modify-preset-editor').value = commands.join('\n');
  document.getElementById('modify-preset-modal').classList.remove('hidden');

  makeModalDraggable('floating-content-window-modify-preset', 'modal-header-modify-preset');
}

// Save the modified preset modal
function saveModifyPreset() {
  const name = document.getElementById('modify-preset-name').value.trim();
  const rawText = document.getElementById('modify-preset-editor').value;

  if (!name) {
    alert("Preset name is required.");
    return;
  }

  const commands = rawText
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0);

  const payload = {
    name: name,
    commands: commands
  };

  fetch('/api/save-new-command-template-preset', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  })
    .then(response => {
      if (!response.ok) throw new Error("Failed to save preset");
      return response.json();
    })
    .then(data => {
      closeModifyPresetModal();
      loadPresets();
    })
    .catch(err => {
      console.error("Error saving preset:", err);
      alert("Failed to save preset.");
    });
}

// Close modify preset modal
function closeModifyPresetModal() {
  document.getElementById('modify-preset-modal').classList.add('hidden');
}

// Validate a queued command template list to add to agent's commands
async function validateCommandTemplatePreset() {
  const queuePreset = document.querySelectorAll('#queue-template-list .queue-item');

  if (!watched.size) {
    alert("No agent selected...")
    return
  }

  try {
    const res = await fetch('/api/command-template-presets');
    if (!res.ok) throw new Error('Failed to fetch presets');
    closeCommandTemplateModal();
    const presets = await res.json();

    for (const item of queuePreset) {
      const selectedPreset = item.querySelector('span')?.textContent;

      if (selectedPreset && presets[selectedPreset]) {
        const commands = presets[selectedPreset];

        for (const cmd of commands) {
          await sendCommand(cmd);
        }
      }
    }
  } catch (err) {
    console.error('Error when retrieving presets', err);
  }
}