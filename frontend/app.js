const API_URL = 'http://localhost:8080';

// Элементы экранов
const authScreen = document.getElementById('auth-screen');
const appScreen = document.getElementById('app-screen');

// Проверяем при запуске, залогинен ли пользователь
document.addEventListener('DOMContentLoaded', () => {
    const token = localStorage.getItem('token');
    if (token) {
        showAppScreen();
    }
});

// Функция переключения на рабочий экран
function showAppScreen() {
    authScreen.classList.add('hidden');
    appScreen.classList.remove('hidden');
    loadExpenses(); // Сразу загружаем траты
}

// ЛОГИКА ВХОДА
document.getElementById('login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const email = document.getElementById('auth-email').value;
    const password = document.getElementById('auth-password').value;
    const errorBlock = document.getElementById('auth-error');

    try {
        const response = await fetch(`${API_URL}/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.message || 'Ошибка входа');
        }

        // Сохраняем токен!
        localStorage.setItem('token', data.token);
        errorBlock.innerText = '';
        showAppScreen();
    } catch (err) {
        errorBlock.innerText = err.message;
    }
});

// ЗАГРУЗКА ТРАТ (С ТОКЕНОМ)
async function loadExpenses() {
    const token = localStorage.getItem('token');
    const list = document.getElementById('expenses-list');
    
    try {
        const response = await fetch(`${API_URL}/expenses`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${token}` // Передаем наш секретный паспорт!
            }
        });

        if (response.status === 401) {
            // Если токен просрочен — выкидываем на экран логина
            logout();
            return;
        }

        const expenses = await response.json();
        list.innerHTML = '';
        
        // Рисуем список трат
        expenses.forEach(exp => {
            const li = document.createElement('li');
            li.innerHTML = `
                <span><strong>${exp.name}</strong> (${exp.category}) — ${exp.price} руб.</span>
                <button onclick="deleteExpense(${exp.id})">❌</button>
            `;
            list.appendChild(li);
        });
    } catch (err) {
        console.error('Ошибка загрузки трат:', err);
    }
}

// ВЫХОД
document.getElementById('btn-logout').addEventListener('click', logout);
function logout() {
    localStorage.removeItem('token');
    authScreen.classList.remove('hidden');
    appScreen.classList.add('hidden');
}