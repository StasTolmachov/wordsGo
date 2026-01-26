//
const API_URL = '/api/v1/wordsGo';
let token = localStorage.getItem('token');
let currentUser = null;

// Pagination State
let wordsPage = 1;
const wordsLimit = 10;
let wordsTotalPages = 1;

// Lesson State
let currentLessonWords = [];
let currentWordIndex = 0;
let hasAttemptedCurrentWord = false;
let lastAnswerWasCorrect = false;

// DOM Elements
const authSection = document.getElementById('auth-section');
const dashboard = document.getElementById('dashboard');
const loginContainer = document.getElementById('login-form-container');
const registerContainer = document.getElementById('register-form-container');
const mainNav = document.getElementById('main-nav');
const toast = document.getElementById('toast');

// Auth Check
if (token) {
    showDashboard();
}

// --- Navigation & UI Helpers ---

function showSection(id) {
    ['search-container', 'words-container', 'lesson-container', 'profile-container'].forEach(s => {
        document.getElementById(s).classList.add('hidden');
    });
    document.getElementById(id).classList.remove('hidden');
}

function showDashboard() {
    authSection.classList.add('hidden');
    dashboard.classList.remove('hidden');
    mainNav.classList.remove('hidden');
    showSection('search-container');
}

function showToast(message, type) {
    toast.textContent = message;
    toast.className = `toast ${type}`;
    toast.classList.remove('hidden');
    setTimeout(() => toast.classList.add('hidden'), 3000);
}

function handleUnauthorized() {
    localStorage.removeItem('token');
    token = null;
    currentUser = null;
    showToast('Session expired. Please login again.', 'error');
    setTimeout(() => location.reload(), 1500);
}

function decodeHTML(html) {
    const txt = document.createElement('textarea');
    txt.innerHTML = html;
    return txt.value;
}

// --- Event Listeners ---

document.getElementById('show-register').onclick = () => {
    loginContainer.classList.add('hidden');
    registerContainer.classList.remove('hidden');
};

document.getElementById('show-login').onclick = () => {
    registerContainer.classList.add('hidden');
    loginContainer.classList.remove('hidden');
};

document.getElementById('logout-btn').onclick = () => {
    localStorage.removeItem('token');
    token = null;
    currentUser = null;
    location.reload();
};

document.getElementById('nav-search').onclick = () => showSection('search-container');

document.getElementById('nav-words').onclick = () => {
    showSection('words-container');
    wordsPage = 1; // Reset to first page
    myWordsSearchQuery = ''; // Reset search
    document.getElementById('my-words-search').value = '';
    loadMyWords();
};

document.getElementById('nav-lesson').onclick = () => {
    showSection('lesson-container');
    startLesson();
};

document.getElementById('exit-lesson-btn').onclick = () => {
    if (confirm('Are you sure you want to exit the lesson?')) {
        showSection('search-container');
    }
};

document.getElementById('nav-profile').onclick = () => {
    showSection('profile-container');
    loadProfile();
};

// Pagination Controls
document.getElementById('words-prev').onclick = () => {
    if (wordsPage > 1) {
        wordsPage--;
        loadMyWords();
    }
};

document.getElementById('words-next').onclick = () => {
    if (wordsPage < wordsTotalPages) {
        wordsPage++;
        loadMyWords();
    }
};


// --- Authentication ---

document.getElementById('login-form').onsubmit = async (e) => {
    e.preventDefault();
    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;

    try {
        const response = await fetch(`${API_URL}/login`, {
            method: 'POST',
            body: JSON.stringify({ email, password }),
            headers: { 'Content-Type': 'application/json' }
        });

        const contentType = response.headers.get("content-type");
        let data;
        if (contentType && contentType.indexOf("application/json") !== -1) {
            data = await response.json();
        } else {
            const text = await response.text();
            console.error("Non-JSON response:", text);
            showToast('Server error: ' + (text || response.statusText), 'error');
            return;
        }

        if (response.ok) {
            token = data.token;
            localStorage.setItem('token', token);
            showDashboard();
            showToast('Login successful!', 'success');
        } else {
            showToast(data.error || 'Login failed', 'error');
        }
    } catch (err) {
        console.error(err);
        showToast('Network error during login', 'error');
    }
};

// Registration Logic
function validatePassword(password) {
    const checks = {
        length: password.length >= 8,
        upper: /[A-Z]/.test(password),
        lower: /[a-z]/.test(password),
        digit: /[0-9]/.test(password),
        special: /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password)
    };
    return checks;
}

function updatePasswordRequirements(password) {
    const checks = validatePassword(password);
    document.getElementById('req-length').className = checks.length ? 'valid' : 'invalid';
    document.getElementById('req-upper').className = checks.upper ? 'valid' : 'invalid';
    document.getElementById('req-lower').className = checks.lower ? 'valid' : 'invalid';
    document.getElementById('req-digit').className = checks.digit ? 'valid' : 'invalid';
    document.getElementById('req-special').className = checks.special ? 'valid' : 'invalid';
    return Object.values(checks).every(v => v);
}

document.getElementById('reg-password').oninput = (e) => {
    updatePasswordRequirements(e.target.value);
};

document.getElementById('register-form').onsubmit = async (e) => {
    e.preventDefault();
    const password = document.getElementById('reg-password').value;

    if (!updatePasswordRequirements(password)) {
        showToast('Password does not meet requirements', 'error');
        return;
    }

    const body = {
        first_name: document.getElementById('reg-firstname').value,
        last_name: document.getElementById('reg-lastname').value,
        email: document.getElementById('reg-email').value,
        password: password,
        source_lang: document.getElementById('reg-sourcelang').value,
        target_lang: document.getElementById('reg-targetlang').value
    };

    try {
        const response = await fetch(`${API_URL}/users`, {
            method: 'POST',
            body: JSON.stringify(body),
            headers: { 'Content-Type': 'application/json' }
        });

        if (response.ok) {
            showToast('Registration successful! Please login.', 'success');
            document.getElementById('show-login').click();
            document.getElementById('register-form').reset();
        } else {
            const contentType = response.headers.get("content-type");
            if (contentType && contentType.indexOf("application/json") !== -1) {
                const data = await response.json();
                showToast(data.error || 'Registration failed', 'error');
            } else {
                const text = await response.text();
                showToast('Registration failed: ' + text, 'error');
            }
        }
    } catch (err) {
        showToast('Network error during registration', 'error');
    }
};

// --- Edit Word Modal Logic ---

const editWordModal = document.getElementById('edit-word-modal');
const editWordForm = document.getElementById('edit-word-form');
const closeModalSpan = document.getElementById('close-modal');
const cancelModalBtn = document.getElementById('cancel-modal-btn');

function openEditModal(word) {
    document.getElementById('edit-word-id').value = word.id;
    document.getElementById('modal-word-title').textContent = `Edit: ${decodeHTML(word.original)}`;
    
    // Pre-fill with custom value if exists, else original
    document.getElementById('edit-transcription').value = word.custom_transcription || word.transcription || '';
    document.getElementById('edit-translation').value = word.custom_translation || word.translation || '';
    document.getElementById('edit-synonyms').value = word.custom_synonyms || word.synonyms || '';

    editWordModal.classList.remove('hidden');
    // Focus the submit button (Add to Learning)
    setTimeout(() => document.getElementById('modal-submit-btn').focus(), 50);
}

function closeEditModal() {
    editWordModal.classList.add('hidden');
    // Restore focus to search input if appropriate
    document.getElementById('search-input').focus();
}

closeModalSpan.onclick = closeEditModal;
cancelModalBtn.onclick = closeEditModal;
window.onclick = (event) => {
    if (event.target == editWordModal) {
        closeEditModal();
    }
};
window.onkeydown = (event) => {
    if (event.key === 'Escape' && !editWordModal.classList.contains('hidden')) {
        closeEditModal();
        return;
    }

    // Lesson handling
    if (!document.getElementById('lesson-container').classList.contains('hidden')) {
        const input = document.getElementById('lesson-answer-input');
        const nextBtn = document.getElementById('next-question');
        const learnBtn = document.getElementById('learn-btn');
        const checkBtn = document.getElementById('check-answer-btn');

        if (event.key === 'Enter') {
            if (input && !input.disabled && checkBtn) {
                event.preventDefault();
                checkBtn.click();
            } else if (nextBtn && !nextBtn.classList.contains('hidden')) {
                event.preventDefault();
                nextBtn.click();
            }
        } else if (event.key === 'Tab') {
            if (learnBtn && !learnBtn.classList.contains('hidden')) {
                event.preventDefault();
                learnBtn.focus();
                learnBtn.click();
            }
        }
    }
};

// Add Enter key support for modal inputs
editWordForm.querySelectorAll('input').forEach(input => {
    input.onkeydown = (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            // Trigger form submission
            editWordForm.requestSubmit();
        }
    };
});

editWordForm.onsubmit = async (e) => {
    e.preventDefault();
    console.log("Submitting edit form...");
    const wordId = document.getElementById('edit-word-id').value;
    console.log("Word ID:", wordId);
    const body = {
        transcription: document.getElementById('edit-transcription').value,
        translation: document.getElementById('edit-translation').value,
        synonyms: document.getElementById('edit-synonyms').value
    };
    console.log("Body:", body);

    try {
        const response = await fetch(`${API_URL}/users/words/${wordId}`, {
            method: 'PATCH',
            body: JSON.stringify(body),
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });

        console.log("Response status:", response.status);

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        if (response.ok) {
            console.log("Update successful");
            showToast('Word details updated!', 'success');
            closeEditModal();
            // Refresh lists if visible
            if (!document.getElementById('search-container').classList.contains('hidden')) {
                // Clear search instead of refreshing it
                document.getElementById('search-input').value = '';
                document.getElementById('search-results').innerHTML = '';
                searchResultsData = [];
                selectedSearchIndex = -1;
            } else if (!document.getElementById('words-container').classList.contains('hidden')) {
                loadMyWords();
            }
        } else {
            const data = await response.json();
            console.error("Update failed:", data);
            showToast(data.error || 'Failed to update word', 'error');
        }
    } catch (err) {
        console.error("Network error:", err);
        showToast('Network error updating word', 'error');
    }
};

// --- Search Dictionary ---

let selectedSearchIndex = -1;
let searchResultsData = [];

document.querySelectorAll('.level-btn').forEach(btn => {
    btn.onclick = () => addWordsByLevel(btn.dataset.level);
});

async function addWordsByLevel(level) {
    if (!confirm(`Add all ${level} words to your learning list?`)) return;

    try {
        const response = await fetch(`${API_URL}/users/words/bulk`, {
            method: 'POST',
            body: JSON.stringify({ level: level }),
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        const data = await response.json();
        if (response.ok) {
            showToast(`Added ${data.count} words!`, 'success');
        } else {
            showToast(data.error || 'Failed to add words', 'error');
        }
    } catch (err) {
        showToast('Network error adding words', 'error');
    }
}

document.getElementById('search-btn').onclick = searchWords;
document.getElementById('search-input').oninput = (e) => {
    const q = e.target.value;
    if (q.length >= 1) {
        searchWords();
    } else {
        document.getElementById('search-results').innerHTML = '';
        searchResultsData = [];
        selectedSearchIndex = -1;
    }
};

document.getElementById('search-input').onkeydown = (e) => {
    const resultsDiv = document.getElementById('search-results');
    const items = resultsDiv.getElementsByClassName('result-item');
    
    if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (selectedSearchIndex < items.length - 1) {
            selectedSearchIndex++;
            highlightSearchResult(items);
        }
    } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (selectedSearchIndex > 0) {
            selectedSearchIndex--;
            highlightSearchResult(items);
        }
    } else if (e.key === 'Enter') {
        e.preventDefault();
        if (selectedSearchIndex >= 0 && selectedSearchIndex < searchResultsData.length) {
            openEditModal(searchResultsData[selectedSearchIndex]);
        } else {
            searchWords();
        }
    }
};

function highlightSearchResult(items) {
    Array.from(items).forEach((item, index) => {
        if (index === selectedSearchIndex) {
            item.classList.add('selected');
            item.scrollIntoView({ block: 'nearest' });
        } else {
            item.classList.remove('selected');
        }
    });
}

async function searchWords() {
    const q = document.getElementById('search-input').value;
    if (!q) {
        document.getElementById('search-results').innerHTML = '';
        return;
    }

    const resultsDiv = document.getElementById('search-results');

    try {
        const response = await fetch(`${API_URL}/dictionary/search?q=${encodeURIComponent(q)}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        const data = await response.json();
        searchResultsData = data || []; // Store for keyboard nav
        selectedSearchIndex = -1;
        resultsDiv.innerHTML = '';

        if (data && data.length > 0) {
            data.forEach((word, index) => {
                const div = document.createElement('div');
                div.className = 'result-item';
                div.dataset.index = index; // Store index for click handling
                
                div.innerHTML = `
                    <div class="word-info">
                        <strong>${decodeHTML(word.original)}</strong>
                        <span>${decodeHTML(word.translation)}</span>
                        ${word.level ? `<small>Level: ${word.level}</small>` : ''}
                    </div>
                    <button class="add-word-btn" data-id="${word.id}">Add</button>
                `;
                
                // Add click listener to the info part for editing
                div.querySelector('.word-info').onclick = () => openEditModal(word);
                
                // Prevent bubbling when clicking add button
                div.querySelector('.add-word-btn').onclick = (e) => {
                    e.stopPropagation();
                    addWord(word.id);
                };

                resultsDiv.appendChild(div);
            });

        } else {
            resultsDiv.innerHTML = '<p>No results found.</p>';
        }
    } catch (err) {
        console.error(err);
        showToast('Error searching words', 'error');
        resultsDiv.innerHTML = '';
    }
}

async function addWord(wordId) {
    console.log("Adding word:", wordId);
    try {
        const response = await fetch(`${API_URL}/users/words`, {
            method: 'POST',
            body: JSON.stringify({ word_id: wordId }),
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });

        console.log("Add word response status:", response.status);

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        if (response.ok) {
            console.log("Word added successfully");
            showToast('Word added to your list!', 'success');
            
            // Clear search
            document.getElementById('search-input').value = '';
            document.getElementById('search-results').innerHTML = '';
            searchResultsData = [];
            selectedSearchIndex = -1;
        } else {
            const data = await response.json();
            console.error("Add word failed:", data);
            showToast(data.error || 'Failed to add word', 'error');
        }
    } catch (err) {
        console.error("Network error adding word:", err);
        showToast('Network error adding word', 'error');
    }
}

// --- My Words (With Pagination and Delete) ---

let myWordsSearchQuery = '';

document.getElementById('my-words-search').oninput = (e) => {
    myWordsSearchQuery = e.target.value;
    wordsPage = 1; // Reset to first page on new search
    loadMyWords();
};

async function loadMyWords() {
    const listDiv = document.getElementById('words-list');
    const paginationDiv = document.getElementById('words-pagination');
    const progressDiv = document.getElementById('progress-container');
    
    listDiv.innerHTML = '<p>Loading...</p>';
    paginationDiv.classList.add('hidden');
    progressDiv.innerHTML = '';

    try {
        const response = await fetch(`${API_URL}/words?limit=${wordsLimit}&page=${wordsPage}&q=${encodeURIComponent(myWordsSearchQuery)}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        const data = await response.json(); // Expected models.ListOfWordsResponse
        
        // Render Progress
        if (data.progress) {
            let progressHtml = `<div class="overall-progress">Overall: ${data.progress.total.toFixed(1)}%</div><div class="level-progress">`;
            for (const [level, percent] of Object.entries(data.progress.by_level)) {
                progressHtml += `<div class="progress-item"><strong>${level}:</strong> ${percent.toFixed(1)}%</div>`;
            }
            progressHtml += '</div>';
            progressDiv.innerHTML = progressHtml;
        }

        listDiv.innerHTML = '';

        const words = data.data || [];
        wordsTotalPages = data.pages || 1;

        if (words.length > 0) {
            words.forEach(word => {
                const div = document.createElement('div');
                div.className = 'word-item';
                
                // Display custom values if present
                const displayTranslation = word.custom_translation || word.translation;
                const displayTranscription = word.custom_transcription || word.transcription;
                // Note: Synonyms are hidden in list view usually, but editable in modal.

                div.innerHTML = `
                    <div class="word-info">
                        <div class="word-header">
                            <strong>${decodeHTML(word.original)}</strong> — <span>${decodeHTML(displayTranslation)}</span>
                        </div>
                        <div class="word-details">
                            ${displayTranscription ? `<span class="tag">/${decodeHTML(displayTranscription)}/</span>` : ''}
                            ${word.pos ? `<span class="tag">${decodeHTML(word.pos)}</span>` : ''}
                            ${word.level ? `<span class="tag level">${decodeHTML(word.level)}</span>` : ''}
                        </div>
                        <div class="word-stats">
                            <span title="Difficulty">Diff: ${word.difficulty_level.toFixed(1)}</span>
                            <span title="Correct Streak">Streak: ${word.correct_streak}</span>
                            <span title="Total Mistakes">Mistakes: ${word.total_mistakes}</span>
                            ${word.is_learned ? '<span class="learned-badge">Learned</span>' : ''}
                        </div>
                    </div>
                    <button class="delete-word-btn" onclick="deleteWord('${word.id}')">Delete</button>
                `;
                
                // Add edit listener
                div.querySelector('.word-info').onclick = () => openEditModal(word);
                
                // Prevent bubbling for delete
                div.querySelector('.delete-word-btn').onclick = (e) => {
                    e.stopPropagation();
                    deleteWord(word.id);
                };

                listDiv.appendChild(div);
            });

            // Update Pagination UI
            document.getElementById('words-page-info').textContent = `Page ${data.page} of ${data.pages}`;
            document.getElementById('words-prev').disabled = data.page <= 1;
            document.getElementById('words-next').disabled = data.page >= data.pages;
            paginationDiv.classList.remove('hidden');

        } else {
            if (myWordsSearchQuery) {
                listDiv.innerHTML = `<p class="search-no-results">No words matching "<strong>${myWordsSearchQuery}</strong>" found in your list.</p>`;
            } else {
                listDiv.innerHTML = `
                    <div class="empty-state">
                        <p>You haven't added any words to your learning list yet.</p>
                        <p>Go to the <strong>Search</strong> page to find and add new words!</p>
                        <button class="primary-btn" onclick="showSection('search-container')">Go to Search</button>
                    </div>
                `;
            }
        }
    } catch (err) {
        console.error(err);
        showToast('Error loading words', 'error');
        listDiv.innerHTML = '<p>Error loading words.</p>';
    }
}

async function deleteWord(wordId) {
    if (!confirm('Are you sure you want to delete this word from your learning list?')) return;

    try {
        const response = await fetch(`${API_URL}/users/words/${wordId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        if (response.ok) {
            showToast('Word deleted', 'success');
            loadMyWords(); // Reload list
        } else {
            showToast('Failed to delete word', 'error');
        }
    } catch (err) {
        console.error(err);
        showToast('Network error deleting word', 'error');
    }
}

// --- Lesson Logic ---

async function startLesson() {
    loadNextLessonBatch();
}

async function loadNextLessonBatch() {
    const contentDiv = document.getElementById('lesson-content');
    
    // Only show loading if we are just starting or have no content
    if (currentLessonWords.length === 0) {
        contentDiv.innerHTML = '<p>Loading lesson...</p>';
    }
    
    document.getElementById('next-question').classList.add('hidden');

    try {
        const response = await fetch(`${API_URL}/lesson/start`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        const data = await response.json();

        if (response.ok && data.words && data.words.length > 0) {
            // Restore content structure in case it was overwritten
            if (!document.getElementById('lesson-question')) {
                contentDiv.innerHTML = `
                    <div id="lesson-question"></div>
                    <div id="lesson-options"></div>
                    <div id="lesson-feedback"></div>
                `;
            }
            
            currentLessonWords = data.words;
            currentWordIndex = 0;
            hasAttemptedCurrentWord = false;
            displayQuestion();
        } else {
            const errorMsg = data.error || 'No words available for lesson. Add some words first!';
            contentDiv.innerHTML = `<p>${errorMsg}</p>`;
        }
    } catch (err) {
        console.error(err);
        showToast('Error starting lesson', 'error');
        contentDiv.innerHTML = '<p>Error starting lesson.</p>';
    }
}

function displayQuestion() {
    const questionDiv = document.getElementById('lesson-question');
    const optionsDiv = document.getElementById('lesson-options');
    const feedbackDiv = document.getElementById('lesson-feedback');
    const nextBtn = document.getElementById('next-question');

    feedbackDiv.innerHTML = '';
    nextBtn.classList.add('hidden');
    optionsDiv.innerHTML = '';

    const currentWord = currentLessonWords[currentWordIndex];
    
    questionDiv.innerHTML = `
        <h3>Translate this word:</h3>
        <div class="lesson-word-display">
            <strong>${decodeHTML(currentWord.translation)}</strong>
        </div>
    `;

    const input = document.createElement('input');
    input.type = 'text';
    input.id = 'lesson-answer-input';
    input.placeholder = 'Type English translation...';
    input.autocomplete = 'off';

    const checkBtn = document.createElement('button');
    checkBtn.textContent = 'Check';
    checkBtn.id = 'check-answer-btn';

    const learnedBtn = document.createElement('button');
    learnedBtn.textContent = 'Mark Learned (Tab)';
    learnedBtn.id = 'learn-btn';
    learnedBtn.classList.add('hidden');
    learnedBtn.style.backgroundColor = 'var(--success-color)';

    optionsDiv.appendChild(input);
    optionsDiv.appendChild(checkBtn);
    optionsDiv.appendChild(learnedBtn);

    input.focus();

    const handleMarkLearned = async () => {
        try {
            const response = await fetch(`${API_URL}/lesson/learned/${currentWord.id}`, {
                method: 'POST',
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (response.ok) {
                showToast('Word marked as learned!', 'success');
                // Move to next word immediately
                lastAnswerWasCorrect = true;
                nextBtn.click();
            }
        } catch (err) {
            console.error(err);
        }
    };

    const handleCheck = () => {
        const userAnswer = input.value.trim().toLowerCase();
        const correctAnswer = decodeHTML(currentWord.original).trim().toLowerCase();
        
        lastAnswerWasCorrect = userAnswer === correctAnswer;

        input.disabled = true;
        checkBtn.disabled = true;

        const detailsHtml = `
            <div class="word-full-details">
                <div class="main-word">${decodeHTML(currentWord.original)} ${currentWord.transcription ? `<span class="transcription">/${decodeHTML(currentWord.transcription)}/</span>` : ''}</div>
                <div class="translation">${decodeHTML(currentWord.translation)}</div>
                <div class="other-info">
                    ${currentWord.pos ? `<span class="tag">${decodeHTML(currentWord.pos)}</span>` : ''}
                    ${currentWord.level ? `<span class="tag level">${decodeHTML(currentWord.level)}</span>` : ''}
                </div>
                ${currentWord.synonyms ? `<div class="synonyms"><strong>Synonyms:</strong> ${decodeHTML(currentWord.synonyms)}</div>` : ''}
                ${currentWord.past_simple_singular || currentWord.past_simple_plural || currentWord.past_participle_singular || currentWord.past_participle_plural ? `
                    <div class="verb-forms">
                        ${currentWord.past_simple_singular ? `<div><strong>Past Simple (s):</strong> ${decodeHTML(currentWord.past_simple_singular)}</div>` : ''}
                        ${currentWord.past_simple_plural ? `<div><strong>Past Simple (p):</strong> ${decodeHTML(currentWord.past_simple_plural)}</div>` : ''}
                        ${currentWord.past_participle_singular ? `<div><strong>Past Participle (s):</strong> ${decodeHTML(currentWord.past_participle_singular)}</div>` : ''}
                        ${currentWord.past_participle_plural ? `<div><strong>Past Participle (p):</strong> ${decodeHTML(currentWord.past_participle_plural)}</div>` : ''}
                    </div>
                ` : ''}
            </div>
        `;

        if (lastAnswerWasCorrect) {
            feedbackDiv.innerHTML = `<span class="correct-text">✓ Correct!</span> ${detailsHtml}`;
            feedbackDiv.className = 'correct';
            learnedBtn.classList.remove('hidden');
            learnedBtn.onclick = handleMarkLearned;
        } else {
            feedbackDiv.innerHTML = `<span class="incorrect-text">✗ Wrong.</span> ${detailsHtml}`;
            feedbackDiv.className = 'incorrect';
        }

        if (!hasAttemptedCurrentWord) {
            submitAnswerToBackend(currentWord.id, lastAnswerWasCorrect);
            hasAttemptedCurrentWord = true;
        }

        nextBtn.classList.remove('hidden');
        if (lastAnswerWasCorrect) {
            nextBtn.textContent = 'Next Word';
        } else {
            nextBtn.textContent = 'Try Again';
        }
        
        nextBtn.focus();
    };

    checkBtn.onclick = handleCheck;
    
    // Set up Next Button Handler
    nextBtn.onclick = () => {
        if (lastAnswerWasCorrect) {
            // Move to next word
            currentWordIndex++;
            hasAttemptedCurrentWord = false; // Reset for new word

            if (currentWordIndex < currentLessonWords.length) {
                displayQuestion();
            } else {
                // Fetch next batch instead of finishing
                showToast('Great job! Loading more words...', 'success');
                loadNextLessonBatch();
            }
        } else {
            // Reload same word, don't increment index
            displayQuestion();
        }
    };
}

async function submitAnswerToBackend(wordId, isCorrect) {
    try {
        const response = await fetch(`${API_URL}/lesson/answer`, {
            method: 'POST',
            body: JSON.stringify({
                word_id: wordId,
                is_correct: isCorrect
            }),
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }
    } catch (err) {
        console.error('Error submitting answer:', err);
    }
}

// --- Profile Logic ---

async function loadProfile() {
    const infoDiv = document.getElementById('user-info');
    infoDiv.innerHTML = '<p>Loading profile...</p>';
    document.getElementById('edit-profile-form-container').classList.add('hidden');

    try {
        const response = await fetch(`${API_URL}/users/me`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        if (response.ok) {
            const user = await response.json();
            currentUser = user;
            infoDiv.innerHTML = `
                <div class="user-details">
                    <p><strong>Name:</strong> ${decodeHTML(user.first_name)} ${decodeHTML(user.last_name)}</p>
                    <p><strong>Email:</strong> ${decodeHTML(user.email)}</p>
                    <p><strong>Role:</strong> ${user.role}</p>
                    <p><strong>Languages:</strong> ${user.source_lang} → ${user.target_lang}</p>
                    <p><strong>Member since:</strong> ${new Date(user.created_at).toLocaleDateString()}</p>
                </div>
            `;
        } else {
            infoDiv.innerHTML = '<p>Failed to load profile.</p>';
        }
    } catch (err) {
        showToast('Error loading profile', 'error');
        infoDiv.innerHTML = '<p>Error loading profile.</p>';
    }
}

// Edit Profile Handlers
const editContainer = document.getElementById('edit-profile-form-container');
const editForm = document.getElementById('edit-profile-form');

document.getElementById('edit-profile-btn').onclick = () => {
    if (!currentUser) return;
    
    // Pre-fill form
    document.getElementById('edit-firstname').value = currentUser.first_name;
    document.getElementById('edit-lastname').value = currentUser.last_name;
    
    editContainer.classList.remove('hidden');
};

document.getElementById('cancel-edit-btn').onclick = () => {
    editContainer.classList.add('hidden');
    editForm.reset();
};

document.addEventListener('DOMContentLoaded', () => {
    const resetProgressBtn = document.getElementById('reset-progress-btn');
    if (resetProgressBtn) {
        resetProgressBtn.addEventListener('click', async (e) => {
            e.preventDefault();
            console.log('Reset progress clicked');
            
            if (!confirm('Are you sure you want to RESET ALL PROGRESS? This will delete all your learning words and stats. This action cannot be undone.')) {
                return;
            }
    
            try {
                const response = await fetch(`${API_URL}/users/progress`, {
                    method: 'DELETE',
                    headers: { 'Authorization': `Bearer ${token}` }
                });
    
                if (response.status === 401) {
                    handleUnauthorized();
                    return;
                }
    
                if (response.ok) {
                    showToast('All progress reset successfully!', 'success');
                    // Refresh dashboard
                    loadMyWords();
                    loadProfile();
                } else {
                    const data = await response.json();
                    showToast(data.error || 'Failed to reset progress', 'error');
                }
            } catch (err) {
                console.error(err);
                showToast('Network error resetting progress', 'error');
            }
        });
    }
});

editForm.onsubmit = async (e) => {
    e.preventDefault();
    if (!currentUser) return;

    const firstName = document.getElementById('edit-firstname').value;
    const lastName = document.getElementById('edit-lastname').value;
    
    const body = {};
    if (firstName) body.first_name = firstName;
    if (lastName) body.last_name = lastName;

    if (Object.keys(body).length === 0) {
        showToast('No changes to update', 'error');
        return;
    }

    try {
        const response = await fetch(`${API_URL}/users/${currentUser.id}`, {
            method: 'PUT',
            body: JSON.stringify(body),
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });

        if (response.status === 401) {
            handleUnauthorized();
            return;
        }

        if (response.ok) {
            const updatedUser = await response.json();
            currentUser = updatedUser; // Update local state
            showToast('Profile updated successfully!', 'success');
            editContainer.classList.add('hidden');
            loadProfile(); // Refresh display
        } else {
            const data = await response.json();
            showToast(data.error || 'Failed to update profile', 'error');
        }
    } catch (err) {
        console.error(err);
        showToast('Error updating profile', 'error');
    }
};
