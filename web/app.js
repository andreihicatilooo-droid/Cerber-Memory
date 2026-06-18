function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = String(str ?? '');
    return div.innerHTML;
}

document.addEventListener('DOMContentLoaded', () => {
    // 1. Tab Navigation
    const navItems = document.querySelectorAll('.nav-item');
    const tabContents = document.querySelectorAll('.tab-content');

    navItems.forEach(item => {
        item.addEventListener('click', () => {
            const targetTab = item.dataset.tab;
            
            navItems.forEach(n => n.classList.remove('active'));
            tabContents.forEach(t => t.classList.remove('active'));
            
            item.classList.add('active');
            const targetElement = document.getElementById(`tab-${targetTab}`);
            if (targetElement) {
                targetElement.classList.add('active');
            }

            // On setting/task tabs active, load corresponding data
            if (targetTab === 'settings') {
                loadConfig();
            } else if (targetTab === 'tasks') {
                loadTasks();
            } else if (targetTab === 'dashboard') {
                loadStats();
            } else if (targetTab === 'mindmap') {
                renderMindmap();
            }
        });
    });

    // 2. Load and Update Stats & Service Statuses
    async function loadStats() {
        try {
            const res = await fetch('/api/stats');
            if (!res.ok) throw new Error('Failed to fetch stats');
            const data = await res.json();

            // Counts
            document.getElementById('count-notes').textContent = data.counts.notes;
            document.getElementById('count-core').textContent = data.counts.core_memory;
            document.getElementById('count-ideas').textContent = data.counts.ideas;
            document.getElementById('count-tasks').textContent = data.counts.tasks;
            document.getElementById('count-docs').textContent = data.counts.documents;
            document.getElementById('count-files').textContent = data.counts.files;

            // Queue status
            document.getElementById('queue-pending').textContent = data.counts.queue_pending;
            document.getElementById('queue-processing').textContent = data.counts.queue_processing;
            document.getElementById('queue-failed').textContent = data.counts.queue_failed;
            document.getElementById('queue-done').textContent = data.counts.queue_done;

            // Badges
            updateServiceBadge('status-qdrant', 'Qdrant: ' + data.services.qdrant, data.services.qdrant);
            updateServiceBadge('status-gcs', 'GCS: ' + data.services.gcs, data.services.gcs);
        } catch (err) {
            console.error('Stats load error:', err);
        }
    }

    function updateServiceBadge(id, text, status) {
        const badge = document.getElementById(id);
        if (!badge) return;

        badge.textContent = text;
        badge.className = 'badge'; // reset

        if (status === 'Online') {
            badge.classList.add('online');
        } else if (status === 'Offline' || status.includes('Error')) {
            badge.classList.add('offline');
        } else {
            badge.classList.add('warning'); // Not Configured
        }
    }

    // Refresh stats button
    document.getElementById('btn-refresh-stats').addEventListener('click', loadStats);

    // 3. Queue Retry Action
    document.getElementById('btn-retry-queue').addEventListener('click', async () => {
        const btn = document.getElementById('btn-retry-queue');
        btn.disabled = true;
        btn.textContent = 'Обработка...';

        try {
            const res = await fetch('/api/queue/retry', { method: 'POST' });
            if (res.ok) {
                alert('Неудачные задачи сброшены обратно в очередь.');
                await loadStats();
            } else {
                alert('Не удалось сбросить очередь.');
            }
        } catch (err) {
            alert('Ошибка сети: ' + err.message);
        } finally {
            btn.disabled = false;
            btn.textContent = 'Повторить попытки ошибок';
        }
    });

    // 4. Ingest Raw Text
    const formIngestText = document.getElementById('form-ingest-text');
    const parseResultsContainer = document.getElementById('parse-results-container');
    const parseResultsElement = document.getElementById('parse-results');

    formIngestText.addEventListener('submit', async (e) => {
        e.preventDefault();
        const submitBtn = document.getElementById('btn-submit-ingest');
        const textInput = document.getElementById('ingest-text');

        submitBtn.disabled = true;
        submitBtn.textContent = 'Импорт и парсинг...';
        parseResultsContainer.classList.add('hidden');

        try {
            const res = await fetch('/api/process', {
                method: 'POST',
                headers: { 'Content-Type': 'text/plain' },
                body: textInput.value
            });

            if (!res.ok) {
                throw new Error(await res.text());
            }

            const data = await res.json();
            parseResultsElement.textContent = JSON.stringify(data, null, 2);
            parseResultsContainer.classList.remove('hidden');
            textInput.value = '';
            alert('Импорт успешно завершен!');
            loadStats();
        } catch (err) {
            alert('Ошибка импорта: ' + err.message);
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Разобрать и импортировать';
        }
    });

    // 5. File Input Label change & File Upload
    const fileInput = document.getElementById('file-input');
    const fileLabelText = document.getElementById('file-label-text');

    fileInput.addEventListener('change', () => {
        if (fileInput.files.length > 0) {
            fileLabelText.textContent = fileInput.files[0].name;
        } else {
            fileLabelText.textContent = 'Выберите файл для загрузки';
        }
    });

    const formUploadFile = document.getElementById('form-upload-file');
    formUploadFile.addEventListener('submit', async (e) => {
        e.preventDefault();
        const submitBtn = document.getElementById('btn-submit-file');
        
        if (fileInput.files.length === 0) {
            alert('Выберите файл!');
            return;
        }

        submitBtn.disabled = true;
        submitBtn.textContent = 'Загрузка...';

        const formData = new FormData();
        formData.append('file', fileInput.files[0]);
        formData.append('project', document.getElementById('upload-project').value);
        formData.append('notebook', document.getElementById('upload-notebook').value);

        try {
            const res = await fetch('/api/files', {
                method: 'POST',
                body: formData
            });

            if (!res.ok) {
                throw new Error(await res.text());
            }

            alert('Файл успешно загружен и привязан!');
            formUploadFile.reset();
            fileLabelText.textContent = 'Выберите файл для загрузки';
            loadStats();
        } catch (err) {
            alert('Ошибка загрузки файла: ' + err.message);
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Загрузить файл';
        }
    });

    // 6. Semantic Search
    const formSearch = document.getElementById('form-search');
    const searchResultsContainer = document.getElementById('search-results-container');

    formSearch.addEventListener('submit', async (e) => {
        e.preventDefault();
        const query = document.getElementById('search-query').value;
        searchResultsContainer.innerHTML = '<div class="glass-card">Поиск...</div>';

        try {
            const res = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
            if (!res.ok) throw new Error('Search request failed');
            const results = await res.json();

            searchResultsContainer.innerHTML = '';
            if (results.length === 0) {
                searchResultsContainer.innerHTML = '<div class="glass-card">Совпадений не найдено.</div>';
                return;
            }

            results.forEach(res => {
                const card = document.createElement('div');
                card.className = 'search-result-card';
                
                const entityType = res.payload.entity_type || 'Unknown';
                const title = res.payload.title || 'Untitled';
                const text = res.payload.text || '';
                const score = res.score;

                card.innerHTML = `
                    <div class="result-header">
                        <span class="result-type">${escapeHtml(entityType)}</span>
                        <span class="result-score">Сходство: ${(score * 100).toFixed(1)}%</span>
                    </div>
                    <div class="result-title">${escapeHtml(title)}</div>
                    <div class="result-text">${escapeHtml(text).replace(/\n/g, '<br>')}</div>
                `;
                searchResultsContainer.appendChild(card);
            });
        } catch (err) {
            searchResultsContainer.innerHTML = `<div class="glass-card" style="color: var(--color-pink);">Ошибка поиска: ${err.message}</div>`;
        }
    });

    // 7. Load and Render Tasks & Subtasks
    async function loadTasks() {
        const container = document.getElementById('tasks-container');
        container.innerHTML = '<div class="glass-card">Загрузка списка задач...</div>';

        try {
            const res = await fetch('/api/tasks');
            if (!res.ok) throw new Error('Failed to load tasks');
            const tasks = await res.json();

            container.innerHTML = '';
            if (tasks.length === 0) {
                container.innerHTML = '<div class="glass-card">Список задач пуст.</div>';
                return;
            }

            tasks.forEach(task => {
                const item = document.createElement('div');
                item.className = 'task-item';

                let priorityClass = 'priority-medium';
                if (task.Priority === 'high') priorityClass = 'priority-high';
                if (task.Priority === 'low') priorityClass = 'priority-low';

                let subtasksHTML = '';
                if (task.Subtasks && task.Subtasks.length > 0) {
                    subtasksHTML = `
                        <div class="subtasks-list">
                            ${task.Subtasks.map(st => `
                                <div class="subtask-item">
                                    <span class="subtask-bullet"></span>
                                    <span>${st.Title} ${st.Details ? `— <em>${st.Details}</em>` : ''} ${st.Tools ? `(Инструменты: <code>${st.Tools}</code>)` : ''}</span>
                                </div>
                            `).join('')}
                        </div>
                    `;
                }

                item.innerHTML = `
                    <div class="task-header">
                        <div class="task-title-group">
                            <span class="task-title">${task.Title}</span>
                            <div class="task-badges">
                                <span class="task-badge ${priorityClass}">${task.Priority}</span>
                                <span class="task-badge" style="background: rgba(255,255,255,0.05); color: #fff;">${task.Status}</span>
                            </div>
                        </div>
                        <div style="font-size: 0.8rem; color: var(--text-muted);">${task.Category || 'General'}</div>
                    </div>
                    ${task.Description ? `<div class="task-desc">${task.Description}</div>` : ''}
                    ${subtasksHTML}
                `;
                container.appendChild(item);
            });
        } catch (err) {
            container.innerHTML = `<div class="glass-card" style="color: var(--color-pink);">Ошибка загрузки задач: ${err.message}</div>`;
        }
    }

    // 8. Load & Save Settings Config
    async function loadConfig() {
        try {
            const res = await fetch('/api/config');
            if (!res.ok) throw new Error('Failed to load config');
            const cfg = await res.json();

            document.getElementById('cfg-gemini-key').placeholder = cfg.gemini_api_key ? 'Сохранено (введите новый для изменения)' : 'API Key не задан';
            document.getElementById('cfg-master-key').placeholder = cfg.cerber_master_key ? 'Сохранено (введите новый для изменения)' : 'Master Key не задан';
            document.getElementById('cfg-qdrant-host').value = cfg.qdrant_host;
            document.getElementById('cfg-qdrant-port').value = cfg.qdrant_port;
            document.getElementById('cfg-gcs-bucket').value = cfg.gcs_bucket_name;
            document.getElementById('cfg-parser-model').value = cfg.gemini_parser_model;
            document.getElementById('cfg-elaborator-model').value = cfg.gemini_elaborator_model;
            document.getElementById('cfg-embedding-model').value = cfg.gemini_embedding_model;
        } catch (err) {
            alert('Ошибка загрузки конфигурации: ' + err.message);
        }
    }

    const formSettings = document.getElementById('form-settings');
    formSettings.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const payload = {
            gemini_api_key: document.getElementById('cfg-gemini-key').value,
            cerber_master_key: document.getElementById('cfg-master-key').value,
            qdrant_host: document.getElementById('cfg-qdrant-host').value,
            qdrant_port: document.getElementById('cfg-qdrant-port').value,
            gcs_bucket_name: document.getElementById('cfg-gcs-bucket').value,
            gemini_parser_model: document.getElementById('cfg-parser-model').value,
            gemini_elaborator_model: document.getElementById('cfg-elaborator-model').value,
            gemini_embedding_model: document.getElementById('cfg-embedding-model').value
        };

        try {
            const res = await fetch('/api/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!res.ok) throw new Error(await res.text());

            alert('Конфигурация успешно сохранена в .env!');
            // clear passwords fields
            document.getElementById('cfg-gemini-key').value = '';
            document.getElementById('cfg-master-key').value = '';
            await loadConfig();
            await loadStats();
        } catch (err) {
            alert('Ошибка сохранения конфигурации: ' + err.message);
        }
    });

    // 9. Vis.js Mindmap / Network Rendering
    let mindmapNetwork = null;

    async function renderMindmap() {
        const container = document.getElementById('mindmap-network');
        if (!container) return;

        container.innerHTML = '<div style="position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); color: #fff;">Загрузка связей...</div>';

        try {
            const res = await fetch('/api/mindmap');
            if (!res.ok) throw new Error('Failed to load mindmap graph data');
            const data = await res.json();

            if (!data.nodes || data.nodes.length === 0) {
                container.innerHTML = '<div style="position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); color: #fff;">Граф памяти пуст. Добавьте новые записи, чтобы увидеть связи.</div>';
                return;
            }

            // Маппинг цветов для различных типов узлов (Modern Dark UI)
            const typeColors = {
                idea: { background: '#2d1b4e', border: '#9b5de5', color: '#fff' },             // Purple
                task: { background: '#1b4d3e', border: '#00bbf9', color: '#fff' },             // Cyan/Green
                project: { background: '#3a232f', border: '#f15bb5', color: '#fff' },          // Pink
                notebook: { background: '#1c3144', border: '#00f5d4', color: '#000' },         // Mint/Teal
                document: { background: '#2b2d42', border: '#8d99ae', color: '#fff' },         // Grey
                file: { background: '#453a22', border: '#fee440', color: '#000' },             // Yellow
                external_resource: { background: '#0a3c36', border: '#38b000', color: '#fff' },// Green
                core_memory: { background: '#4f1a21', border: '#ff0054', color: '#fff' }       // Red
            };

            const nodes = data.nodes.map(n => {
                const colors = typeColors[n.type] || { background: '#222', border: '#777', color: '#fff' };
                return {
                    id: n.id,
                    label: `<b>${n.type.toUpperCase()}</b>\n${n.title}`,
                    font: { multi: 'html', color: colors.color },
                    color: {
                        background: colors.background,
                        border: colors.border,
                        highlight: {
                            background: colors.border,
                            border: '#ffffff'
                        }
                    },
                    shape: 'box',
                    margin: 10,
                    borderWidth: 2,
                    borderRadius: 8
                };
            });

            const edges = data.edges.map(e => {
                return {
                    from: e.source,
                    to: e.target,
                    label: e.label,
                    font: { size: 10, color: '#8d99ae', align: 'horizontal' },
                    color: { color: 'rgba(255,255,255,0.2)', highlight: '#ffffff' },
                    arrows: 'to',
                    width: 1.5
                };
            });

            const visData = {
                nodes: new vis.DataSet(nodes),
                edges: new vis.DataSet(edges)
            };

            const options = {
                physics: {
                    barnesHut: {
                        gravitationalConstant: -3000,
                        centralGravity: 0.3,
                        springLength: 95,
                        springConstant: 0.04,
                        damping: 0.09
                    },
                    stabilization: { iterations: 150 }
                },
                interaction: {
                    hover: true,
                    tooltipDelay: 200,
                    navigationButtons: true,
                    keyboard: true
                }
            };

            container.innerHTML = '';
            mindmapNetwork = new vis.Network(container, visData, options);

        } catch (err) {
            console.error('Mindmap render error:', err);
            container.innerHTML = `<div style="position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); color: var(--color-pink);">Ошибка рендеринга графа: ${err.message}</div>`;
        }
    }

    const btnRefreshMindmap = document.getElementById('btn-refresh-mindmap');
    if (btnRefreshMindmap) {
        btnRefreshMindmap.addEventListener('click', renderMindmap);
    }

    // Initial load on startup
    loadStats();
});
