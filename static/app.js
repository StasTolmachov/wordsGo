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
        const el = document.getElementById(s);
        if (el) el.classList.add('hidden');
    });
    const target = document.getElementById(id);
    if (target) target.classList.remove('hidden');
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
    wordsPage = 1;
    myWordsSearchQuery = '';
    const searchInput = document.getElementById('my-words-search');
    if (searchInput) searchInput.value = '';
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

// --- Pagination Controls ---
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

// --- GLOBAL EVENT DELEGATION FOR LESSON (FIXED) ---
// Мы слушаем клики на всем body и перехватываем клик по кнопке #next-question
document.body.addEventListener('click', (e) => {
    // Проверяем, кликнули ли по кнопке "Next/Try Again"
    if (e.target && e.target.id === 'next-question') {
        console.log('Next button clicked via delegation!');

        const nextBtn = document.getElementById('next-question');
        nextBtn.classList.add('hidden'); // Сразу скрываем, чтобы не кликнули дважды

        if (lastAnswerWasCorrect) {
            console.log('Answer was correct, advancing index...');
            // Логика "Следующее слово"
            currentWordIndex++;

            if (currentLessonWords && currentWordIndex < currentLessonWords.length) {
                displayQuestion();
            } else {
                console.log('Batch finished, loading new words...');
                showToast('Great job! Loading more words...', 'success');
                loadNextLessonBatch();
            }
        } else {
            console.log('Answer was incorrect, retrying same word...');
            // Логика "Попробовать еще раз"
            displayQuestion();
        }
    }
});


// --- Authentication (Login/Register) ---
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

        const data = await response.json();
        if (response.ok) {
            token = data.token;
            localStorage.setItem('token', token);
            showDashboard();
            showToast('Login successful!', 'success');
        } else {
            showToast(data.error || 'Login failed', 'error');
        }
    } catch (err) {
        showToast('Network error during login', 'error');
    }
};

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
            const data = await response.json();
            showToast(data.error || 'Registration failed', 'error');
        }
    } catch (err) { showToast('Network error during registration', 'error'); }
};


// --- Edit Word Modal ---
const editWordModal = document.getElementById('edit-word-modal');
const editWordForm = document.getElementById('edit-word-form');
const closeModalSpan = document.getElementById('close-modal');
const cancelModalBtn = document.getElementById('cancel-modal-btn');

function openEditModal(word) {
    document.getElementById('edit-word-id').value = word.id;
    document.getElementById('modal-word-title').textContent = `Edit: ${decodeHTML(word.original)}`;
    document.getElementById('edit-transcription').value = word.custom_transcription || word.transcription || '';
    document.getElementById('edit-translation').value = word.custom_translation || word.translation || '';
    document.getElementById('edit-synonyms').value = word.custom_synonyms || word.synonyms || '';
    editWordModal.classList.remove('hidden');
    setTimeout(() => document.getElementById('modal-submit-btn').focus(), 50);
}

function closeEditModal() {
    editWordModal.classList.add('hidden');
}

closeModalSpan.onclick = closeEditModal;
cancelModalBtn.onclick = closeEditModal;

editWordForm.onsubmit = async (e) => {
    e.preventDefault();
    const wordId = document.getElementById('edit-word-id').value;
    const body = {
        transcription: document.getElementById('edit-transcription').value,
        translation: document.getElementById('edit-translation').value,
        synonyms: document.getElementById('edit-synonyms').value
    };
    try {
        const response = await fetch(`${API_URL}/users/words/${wordId}`, {
            method: 'PATCH',
            body: JSON.stringify(body),
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });
        if (response.ok) {
            showToast('Word updated!', 'success');
            closeEditModal();
            if (!document.getElementById('search-container').classList.contains('hidden')) searchWords();
            else loadMyWords();
        }
    } catch (err) { showToast('Network error updating word', 'error'); }
};

// --- Search Dictionary ---
let searchResultsData = [];
document.querySelectorAll('.level-btn').forEach(btn => {
    btn.onclick = () => addWordsByLevel(btn.dataset.level);
});

async function addWordsByLevel(level) {
    if (!confirm(`Add all ${level} words?`)) return;
    try {
        const response = await fetch(`${API_URL}/users/words/bulk`, {
            method: 'POST',
            body: JSON.stringify({ level: level }),
            headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` }
        });
        const data = await response.json();
        if (response.ok) showToast(`Added ${data.count} words!`, 'success');
    } catch (err) { showToast('Error adding words', 'error'); }
}

document.getElementById('search-btn').onclick = searchWords;
document.getElementById('search-input').oninput = (e) => { if (e.target.value.length >= 1) searchWords(); };

async function searchWords() {
    const q = document.getElementById('search-input').value;
    const resultsDiv = document.getElementById('search-results');
    if (!q) { resultsDiv.innerHTML = ''; return; }
    try {
        const response = await fetch(`${API_URL}/dictionary/search?q=${encodeURIComponent(q)}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const data = await response.json();
        searchResultsData = data || [];
        resultsDiv.innerHTML = '';
        if (data && data.length > 0) {
            data.forEach((word, index) => {
                const div = document.createElement('div');
                div.className = 'result-item';
                div.innerHTML = `
                    <div class="word-info">
                        <strong>${decodeHTML(word.original)}</strong>
                        <span>${decodeHTML(word.translation)}</span>
                    </div>
                    <button class="add-word-btn">Add</button>
                `;
                div.querySelector('.word-info').onclick = () => openEditModal(word);
                div.querySelector('.add-word-btn').onclick = (e) => { e.stopPropagation(); addWord(word.id); };
                resultsDiv.appendChild(div);
            });
        } else { resultsDiv.innerHTML = '<p>No results found.</p>'; }
    } catch (err) { resultsDiv.innerHTML = ''; }
}

async function addWord(wordId) {
    try {
        const response = await fetch(`${API_URL}/users/words`, {
            method: 'POST',
            body: JSON.stringify({ word_id: wordId }),
            headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` }
        });
        if (response.ok) showToast('Word added!', 'success');
    } catch (err) { showToast('Error adding word', 'error'); }
}

// --- My Words ---
let myWordsSearchQuery = '';
document.getElementById('my-words-search').oninput = (e) => {
    myWordsSearchQuery = e.target.value;
    wordsPage = 1;
    loadMyWords();
};

async function loadMyWords() {
    const listDiv = document.getElementById('words-list');
    const paginationDiv = document.getElementById('words-pagination');
    const progressDiv = document.getElementById('progress-container');
    listDiv.innerHTML = '<p>Loading...</p>';
    try {
        const response = await fetch(`${API_URL}/words?limit=${wordsLimit}&page=${wordsPage}&q=${encodeURIComponent(myWordsSearchQuery)}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const data = await response.json();
        if (data.progress) {
            let p = `<div class="overall-progress">Overall: ${data.progress.total.toFixed(1)}%</div><div class="level-progress">`;
            for (const [lvl, pct] of Object.entries(data.progress.by_level)) {
                p += `<div class="progress-item"><strong>${lvl}:</strong> ${pct.toFixed(1)}%</div>`;
            }
            progressDiv.innerHTML = p + '</div>';
        }
        listDiv.innerHTML = '';
        const words = data.data || [];
        wordsTotalPages = data.pages || 1;
        if (words.length > 0) {
            words.forEach(word => {
                const div = document.createElement('div');
                div.className = 'word-item';
                div.innerHTML = `
                    <div class="word-info">
                        <div class="word-header"><strong>${decodeHTML(word.original)}</strong> — ${decodeHTML(word.custom_translation || word.translation)}</div>
                    </div>
                    <button class="delete-word-btn">Delete</button>
                `;
                div.querySelector('.word-info').onclick = () => openEditModal(word);
                div.querySelector('.delete-word-btn').onclick = (e) => { e.stopPropagation(); deleteWord(word.id); };
                listDiv.appendChild(div);
            });
            document.getElementById('words-page-info').textContent = `Page ${data.page} of ${data.pages}`;
            document.getElementById('words-prev').disabled = data.page <= 1;
            document.getElementById('words-next').disabled = data.page >= data.pages;
            paginationDiv.classList.remove('hidden');
        } else { listDiv.innerHTML = '<p>No words found.</p>'; }
    } catch (err) { listDiv.innerHTML = '<p>Error loading.</p>'; }
}

async function deleteWord(wordId) {
    if (!confirm('Delete word?')) return;
    try {
        const response = await fetch(`${API_URL}/users/words/${wordId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (response.ok) { showToast('Deleted', 'success'); loadMyWords(); }
    } catch (err) { showToast('Error deleting', 'error'); }
}

// --- Lesson Logic (FINAL FIX WITH DELEGATION) ---

async function startLesson() {
    currentLessonWords = [];
    currentWordIndex = 0;
    loadNextLessonBatch();
}

async function loadNextLessonBatch() {
    const statusDiv = document.getElementById('lesson-status');
    const questionDiv = document.getElementById('lesson-question');
    const optionsDiv = document.getElementById('lesson-options');
    const feedbackDiv = document.getElementById('lesson-feedback');
    const nextBtn = document.getElementById('next-question');

    // Очищаем старые данные. НЕ ИСПОЛЬЗУЕМ innerHTML НА РОДИТЕЛЯХ
    if (statusDiv) statusDiv.innerHTML = '<p>Loading lesson words...</p>';
    if (questionDiv) questionDiv.innerHTML = '';
    if (optionsDiv) optionsDiv.innerHTML = '';
    if (feedbackDiv) feedbackDiv.innerHTML = '';

    // Скрываем кнопку
    if (nextBtn) nextBtn.classList.add('hidden');

    try {
        const response = await fetch(`${API_URL}/lesson/start`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const data = await response.json();

        console.log("Lesson data loaded:", data);

        if (response.ok && data.words && data.words.length > 0) {
            if (statusDiv) statusDiv.innerHTML = '';
            currentLessonWords = data.words;
            currentWordIndex = 0;
            displayQuestion();
        } else {
            if (statusDiv) statusDiv.innerHTML = `<p>${data.error || 'Add more words to start a lesson!'}</p>`;
        }
    } catch (err) {
        console.error("Lesson load error:", err);
        if (statusDiv) statusDiv.innerHTML = '<p>Network error starting lesson.</p>';
    }
}

function displayQuestion() {
    console.log("Displaying question. Index:", currentWordIndex);

    const questionDiv = document.getElementById('lesson-question');
    const optionsDiv = document.getElementById('lesson-options');
    const feedbackDiv = document.getElementById('lesson-feedback');
    const nextBtn = document.getElementById('next-question');

    // Сброс состояния UI
    if (feedbackDiv) {
        feedbackDiv.innerHTML = '';
        feedbackDiv.className = '';
    }

    // Скрываем кнопку, пока пользователь не проверит ответ
    if (nextBtn) nextBtn.classList.add('hidden');

    if (optionsDiv) optionsDiv.innerHTML = '';

    hasAttemptedCurrentWord = false;

    const currentWord = currentLessonWords[currentWordIndex];
    if (!currentWord) {
        console.log("No word found at index, loading next batch...");
        loadNextLessonBatch();
        return;
    }

    if (questionDiv) {
        questionDiv.innerHTML = `
            <h3>Translate this word:</h3>
            <div class="lesson-word-display"><strong>${decodeHTML(currentWord.translation)}</strong></div>
        `;
    }

    const input = document.createElement('input');
    input.type = 'text';
    input.id = 'lesson-answer-input';
    input.placeholder = 'Type English word...';
    input.autocomplete = 'off';

    const checkBtn = document.createElement('button');
    checkBtn.textContent = 'Check';
    checkBtn.className = 'primary-btn';
    checkBtn.id = 'check-answer-btn'; // ID для ясности

    optionsDiv.appendChild(input);
    optionsDiv.appendChild(checkBtn);
    input.focus();

    // Локальная функция проверки
    const handleCheck = () => {
        console.log("Checking answer...");
        const userAnswer = input.value.trim().toLowerCase();
        const correctAnswer = decodeHTML(currentWord.original).trim().toLowerCase();

        lastAnswerWasCorrect = (userAnswer === correctAnswer);
        console.log("Is Correct:", lastAnswerWasCorrect);

        input.disabled = true;
        checkBtn.disabled = true;

        if (lastAnswerWasCorrect) {
            feedbackDiv.innerHTML = '<span class="correct-text">✓ Correct!</span>';
            feedbackDiv.className = 'correct';
            if (nextBtn) nextBtn.textContent = 'Next Word';
        } else {
            feedbackDiv.innerHTML = `<span class="incorrect-text">✗ Incorrect.</span> Correct answer: <strong>${decodeHTML(currentWord.original)}</strong>`;
            feedbackDiv.className = 'incorrect';
            if (nextBtn) nextBtn.textContent = 'Try Again';
        }

        if (!hasAttemptedCurrentWord) {
            submitAnswerToBackend(currentWord.id, lastAnswerWasCorrect);
            hasAttemptedCurrentWord = true;
        }

        // Показываем кнопку. Клик по ней обработает ГЛОБАЛЬНЫЙ делегат (вверху файла)
        if (nextBtn) {
            nextBtn.classList.remove('hidden');
            nextBtn.focus();
        }
    };

    checkBtn.onclick = handleCheck;
    input.onkeydown = (e) => { if (e.key === 'Enter') handleCheck(); };
}

async function submitAnswerToBackend(wordId, isCorrect) {
    try {
        await fetch(`${API_URL}/lesson/answer`, {
            method: 'POST',
            body: JSON.stringify({ word_id: wordId, is_correct: isCorrect }),
            headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` }
        });
    } catch (err) { console.error('Backend error', err); }
}

// --- Profile ---
async function loadProfile() {
    const infoDiv = document.getElementById('user-info');
    try {
        const response = await fetch(`${API_URL}/users/me`, { headers: { 'Authorization': `Bearer ${token}` } });
        const user = await response.json();
        currentUser = user;
        infoDiv.innerHTML = `
            <p><strong>Name:</strong> ${decodeHTML(user.first_name)} ${decodeHTML(user.last_name)}</p>
            <p><strong>Email:</strong> ${decodeHTML(user.email)}</p>
        `;
    } catch (err) { infoDiv.innerHTML = 'Error loading.'; }
}