const API_URL = '';

// Выносим функции в глобальную область или привязываем к window,
// чтобы инлайновые обработчики (onclick) могли их найти.
window.deleteExpense = async function(id) {
    const token = localStorage.getItem('token');
    if (!id || !confirm('Удалить эту трату?')) return;

    try {
        console.log(`🗑️ Отправка запроса DELETE на /expenses/${id}`);
        const response = await fetch(`${API_URL}/expenses/${id}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.status === 401) return logout();
        if (!response.ok) throw new Error('Не удалось удалить трату сервером');

        console.log(`✅ Трата ${id} успешно удалена!`);
        loadExpenses();
        loadTotal();
    } catch (err) {
        console.error('❌ Ошибка удаления:', err);
        alert(err.message);
    }
};

document.addEventListener('DOMContentLoaded', () => {
    console.log("🚀 DOM дерево загружено. Инициализация приложения...");

    const loginForm = document.getElementById('login-form');
    const btnRegister = document.getElementById('btn-register');
    const expenseForm = document.getElementById('expense-form');
    const btnLogout = document.getElementById('btn-logout');

    if (!loginForm) {
        console.error("🔴 КРИТИЧЕСКАЯ ОШИБКА: Не найден элемент с id='login-form' в HTML!");
        return;
    }

    const token = localStorage.getItem('token');
    if (token) {
        console.log("🎫 Найдена сессия (токен). Переходим в личный кабинет...");
        showAppScreen();
    }

    // Авторизация
    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = document.getElementById('auth-email').value;
        const password = document.getElementById('auth-password').value;
        const authError = document.getElementById('auth-error');

        try {
            const response = await fetch(`${API_URL}/login`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });

            const data = await response.json();
            if (!response.ok) throw new Error(data.message || 'Неверный email или пароль');

            localStorage.setItem('token', data.token);
            if (authError) authError.innerText = '';
            showAppScreen();
        } catch (err) {
            console.error("❌ Ошибка входа:", err);
            if (authError) {
                authError.innerText = err.message;
                authError.style.color = '#dc3545';
            }
        }
    });

    // Регистрация
    if (btnRegister) {
        btnRegister.addEventListener('click', async () => {
            const email = document.getElementById('auth-email').value;
            const password = document.getElementById('auth-password').value;
            const authError = document.getElementById('auth-error');

            if (!email || !password) {
                if (authError) authError.innerText = 'Заполните поля для регистрации';
                return;
            }

            try {
                const response = await fetch(`${API_URL}/register`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ email, password })
                });

                const data = await response.json();
                if (!response.ok) throw new Error(data.message || 'Ошибка регистрации');

                if (authError) {
                    authError.innerText = 'Успешно! Теперь нажмите "Войти"';
                    authError.style.color = '#28a745';
                }
            } catch (err) {
                console.error("❌ Ошибка регистрации:", err);
                if (authError) {
                    authError.innerText = err.message;
                    authError.style.color = '#dc3545';
                }
            }
        });
    }

    // Добавление траты
    if (expenseForm) {
        expenseForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const token = localStorage.getItem('token');
            const name = document.getElementById('exp-name').value;
            const price = parseFloat(document.getElementById('exp-price').value);
            const category = document.getElementById('exp-category').value;

            try {
                console.log("📤 Отправка новой траты на сервер...");
                const response = await fetch(`${API_URL}/expenses`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${token}`
                    },
                    body: JSON.stringify({ name, price, category })
                });

                if (response.status === 401) return logout();
                if (!response.ok) throw new Error('Сервер отклонил добавление траты');

                expenseForm.reset();
                console.log("✅ Трата успешно добавлена. Обновляем списки...");
                loadExpenses();
                loadTotal();
            } catch (err) {
                console.error("❌ Ошибка добавления траты:", err);
                alert(err.message);
            }
        });
    }

    if (btnLogout) {
        btnLogout.addEventListener('click', logout);
    }
});

// Функции управления экранами
function showAppScreen() {
    const authScreen = document.getElementById('auth-screen');
    const appScreen = document.getElementById('app-screen');
    if (authScreen) authScreen.classList.add('hidden');
    if (appScreen) appScreen.classList.remove('hidden');
    loadExpenses();
    loadTotal();
}

function logout() {
    localStorage.removeItem('token');
    const authScreen = document.getElementById('auth-screen');
    const appScreen = document.getElementById('app-screen');
    if (authScreen) authScreen.classList.remove('hidden');
    if (appScreen) appScreen.classList.add('hidden');
    console.log("🚪 Выход из системы выполнен.");
}

// Загрузка данных
async function loadExpenses() {
    const token = localStorage.getItem('token');
    const list = document.getElementById('expenses-list');
    if (!list) return;

    try {
        console.log("🔄 Запрос списка трат с бэкенда...");
        const response = await fetch(`${API_URL}/expenses`, {
            method: 'GET',
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (response.status === 401) return logout();
        if (!response.ok) {
            list.innerHTML = `<li style="color: red;">Ошибка сервера (Статус ${response.status})</li>`;
            return;
        }

        const expenses = await response.json();
        console.log("📦 Ответ сервера на /expenses:", expenses);
        list.innerHTML = '';

        if (!expenses || !Array.isArray(expenses) || expenses.length === 0) {
            list.innerHTML = '<li style="justify-content: center; color: #6c757d;">Трат пока нет</li>';
            return;
        }

        expenses.forEach(exp => {
            const id = exp.ID ?? exp.id;
            const name = exp.Name ?? exp.name ?? 'Без названия';
            const category = exp.Category ?? exp.category ?? 'Разное';
            const price = exp.Price ?? exp.price ?? 0;

            const li = document.createElement('li');
            li.innerHTML = `<span><strong>${name}</strong> <small style="color:#6c757d">(${category})</small> — ${price} руб.</span>
                            <button onclick="deleteExpense(${id})" style="background:none; border:none; cursor:pointer; font-size:16px;">❌</button>`;
            list.appendChild(li);
        });
    } catch (err) {
        console.error('🔴 Критическая ошибка в loadExpenses:', err);
        list.innerHTML = `<li style="color: #dc3545;">Не удалось отобразить список трат</li>`;
    }
}

async function loadTotal() {
    const token = localStorage.getItem('token');
    const totalBlock = document.getElementById('total-amount');
    if (!totalBlock) return;

    try {
        const response = await fetch(`${API_URL}/total`, {
            method: 'GET',
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (!response.ok) {
            totalBlock.innerText = "Общая сумма: ошибка сервера";
            return;
        }

        const data = await response.json();
        const totalValue = data.total_price ?? 0;
        totalBlock.innerText = `Общая сумма: ${totalValue} руб.`;
    } catch (err) {
        console.error('🔴 Критическая ошибка в loadTotal:', err);
        totalBlock.innerText = `Общая сумма: ошибка расчета`;
    }
}
