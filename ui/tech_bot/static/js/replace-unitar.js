// Блок 1: Обрезка пробелов в textarea
document.addEventListener("DOMContentLoaded", function () {
  // Ждем, пока DOM полностью загрузится
  const textareas = document.querySelectorAll(".autoResizeTextarea");
  // Выбираем все textarea с классом 'autoResizeTextarea'
  textareas.forEach((textarea) => {
    // Для каждого найденного textarea
    textarea.value = textarea.value.trim();
    // Обрезаем пробелы в начале и конце текста
  });
});

// Блок 2: Автоматическое изменение размера textarea
document.addEventListener("DOMContentLoaded", function () {
  // Снова ждем загрузки DOM
  const textareas = document.querySelectorAll(".autoResizeTextarea");
  // Выбираем все textarea с классом 'autoResizeTextarea'

  function autoResize() {
    // Функция для автоматического изменения размера
    this.style.height = "auto"; // Сбрасываем высоту
    let newHeight = this.scrollHeight; // Получаем высоту содержимого
    let maxHeight = parseInt(window.getComputedStyle(this).maxHeight);
    // Получаем максимальную высоту из CSS

    if (newHeight > maxHeight) {
      // Если новая высота больше максимальной
      this.style.height = maxHeight + "px"; // Устанавливаем максимальную высоту
      this.style.overflowY = "auto"; // Включаем вертикальную прокрутку
    } else {
      // Если новая высота не превышает максимальную
      this.style.height = newHeight + "px"; // Устанавливаем новую высоту
      this.style.overflowY = "hidden"; // Скрываем прокрутку
    }
  }

  textareas.forEach((textarea) => {
    // Для каждого textarea
    textarea.addEventListener("input", autoResize);
    // Вызываем функцию при вводе текста

    autoResize.call(textarea);
    // Вызываем функцию при загрузке страницы (если есть начальный текст)
  });
});
// Блок 3: Выпадающие списки
document.addEventListener("DOMContentLoaded", function () {
  const customSelects = document.querySelectorAll(".custom-select");

  function initializeSelect(select) {
    const selected = select.querySelector(".select-selected");
    const search = select.querySelector(".select-search");
    const items = select.querySelector(".select-items");

    function handleItemClick(item) {
      const previousValue = selected.textContent;
      selected.textContent = item.textContent;
      closeDropdown(select);

      if (previousValue !== item.textContent) {
        if (selected.id === "subject" || selected.id === "replace-format") {
          const event = new CustomEvent(`${selected.id}Changed`, {
            detail: { value: item.textContent },
          });
          selected.dispatchEvent(event);
        }
      }
    }

    // Используем делегирование событий на уровне items
    items.addEventListener("click", function (e) {
      if (e.target.tagName === "LI") {
        handleItemClick(e.target);
      }
    });

    selected.addEventListener("click", function (e) {
      e.stopPropagation();
      closeAllDropdowns();
      items.style.display = "block";
      selected.style.display = "none";
      search.style.display = "block";
      search.focus();
      search.value = "";
      filterItems("");
    });

    search.addEventListener("input", function () {
      filterItems(this.value.toLowerCase());
    });

    function filterItems(filter) {
      const listItems = items.querySelectorAll("li");
      listItems.forEach((item) => {
        if (item.textContent.toLowerCase().indexOf(filter) > -1) {
          item.style.display = "";
        } else {
          item.style.display = "none";
        }
      });
    }
  }

  customSelects.forEach(initializeSelect);

  document.addEventListener("click", closeAllDropdowns);

  function closeDropdown(select) {
    const selected = select.querySelector(".select-selected");
    const search = select.querySelector(".select-search");
    const items = select.querySelector(".select-items");
    items.style.display = "none";
    selected.style.display = "block";
    search.style.display = "none";
    search.value = "";
    search.blur();
  }

  function closeAllDropdowns() {
    customSelects.forEach(closeDropdown);
  }

  window.initializeCustomSelect = initializeSelect;
});
