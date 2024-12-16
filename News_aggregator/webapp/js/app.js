async function loadNews() {
    const newsContainer = document.getElementById("news-container");

    try {
        // Запрашиваем последние 10 новостей
        const response = await fetch('/news/10');
        
        if (!response.ok) {
            throw new Error("Ошибка при загрузке новостей");
        }

        const posts = await response.json();

        // Очищаем контейнер перед добавлением новых карточек
        newsContainer.innerHTML = '';

        // Генерируем HTML для каждой новости
        posts.forEach(post => {
            const newsCard = document.createElement("div");
            newsCard.classList.add("news-card");

            newsCard.innerHTML = `
                <h2><a href="${post.Link}" target="_blank">${post.Title}</a></h2>
                <p>${post.Content}</p>
                <div class="pubdate">${post.PubDate}</div>
            `;

            newsContainer.appendChild(newsCard);
        });
    } catch (error) {
        console.error("Ошибка загрузки новостей:", error);
        newsContainer.innerHTML = `<p>Не удалось загрузить новости. Попробуйте обновить страницу позже.</p>`;
    }
}

// Загружаем новости при загрузке страницы
window.onload = loadNews;