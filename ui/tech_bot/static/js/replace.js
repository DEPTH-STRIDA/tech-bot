const buttons = {
  new_form: "new-form-button", // Новая заявка
  history: "history-button", // История
  send: "send-button", // Кнопка отправки
  switcher: "switcher", // Замена/ Перенос
  replace_button_style: "replace-button-style",
  transfer_button_style: "transfer-button-style",
};
const main_sections = {
  form_container: "form-container", // Контейнер с формой
  history_container: "history-container", // Контейнер с карточками заявок
};
const lesson_info = {
  // Дата и время урока
  date: "lesson-date",
  time: "lesson-time",
  // Время переноса
  transfer_time_container: "transfer-time-container",
  transfer_time_input: "transfer-time",
  // Комментарий
  comment_container: "comment-container",
  comment_input: "comment",
  // Формат замены
  replace_format: "replace-format",
  subject: "subject",
  group_number: "group-number",
  module: "module",
  lesson: "lesson",
  teacher: "teacher",
  team_leader: "teamleader",
  reason: "reason",
  reason_title: "reason-title",
  // Ссылка
  link_container: "link-container",
  link_input: "link",
  lint_title: "lint-title",
  // Обычная важная инфа
  imp_info_container: "imp-info-container",
  imp_info_input: "imp-info",
  imp_info_input_2: "imp-info-2",
  // Наставничество - важная инфа
  imp_info_mentor_container: "imp-info-mentor-container",
  mentoring_inf_1: "mentoring-inf-1",
  mentoring_inf_2: "mentoring-inf-2",
  mentoring_inf_3: "mentoring-inf-3",
};
const error_alert = {
  alert_container: "alert-container",
  alert_close_button: "alert-close-button",
  error_description: "error-description",
  loading: "loading",
  title: "title",
};

// new; history; edit
let currentMode = "new";

let actualFormType = "transfer";
let linkIsEnabled = true;
let isMentoring = false;
let dropListReplacValue = "Формат";
let dropListTransferValue = "Формат";
let currentHistoryID = -1;

let SAVE = {};
let history;

let tg, initData;

// main
document.addEventListener("DOMContentLoaded", function () {
  // Иницилизация переменных
  const allObjects = { buttons, main_sections, lesson_info, error_alert };
  convertIdsToElements(allObjects);
  // Иницилизация телеграмм
  const result = initializeTg(5, 2000); // 5 попыток с интервалом в 2 секунды
  if (result) {
    console.log("Telegram Web App успешно инициализирован");
    tg = result.tg;
    initData = result.initData;
    // Дальнейшая логика с использованием tg и initData
  } else {
    console.error("Ошибка инициализации Telegram Web App");
    showAlert("Пожалуйста, запустите форму через телеграмм", 15, true);
    return;
  }
  // При загрузке страницы, устанавливать вид "новой заявки"
  setVisible("form-container");
  switchFormType();
  setNewSubject("");
  setButtonsHandlers();
  getSelectData();
  saveForm();
  setNewEditVisible("new");

  // showAlert("Заявка скоро появится в телеграмм",10,false);

  // showAlert("тесттестестт естесттесттест есттесттестестт есттестесттест");
});

/**
 * setButtonHandlers устанавливает обработчики кнопок
 */
function setButtonsHandlers() {
  // Новая форма
  buttons.new_form.addEventListener("click", function () {
    // Если переход осуществлен из состояния "заполнения формы", то сохраняем форму
    if (currentMode == "edit") {
      saveForm("edit");
    } else if (currentMode == "new") {
      saveForm("new");
    }

    setVisible("form-container");
    buttons.new_form.classList.add("white-line");
    buttons.history.classList.remove("white-line");

    currentMode = "new";
    setNewEditVisible("new"); // Изменение текста кнопок
    loadForm(currentMode);
  });

  // История
  buttons.history.addEventListener("click", function () {
    // Если переход осуществлен из состояния "заполнения формы", то сохраняем форму
    if (currentMode == "edit") {
      saveForm("edit");
    } else if (currentMode == "new") {
      saveForm("new");
    }

    setVisible("history-container");
    buttons.history.classList.add("white-line");
    buttons.new_form.classList.remove("white-line");

    getHistoryData(false);
    currentMode = "history";
    setNewEditVisible("new"); // Изменение текста кнопок

    ////////////////////////////////////////////////////////////////////////////////////
    // let i = 0;
    // history.forEach((item) => {
    //   i++;
    //   console.log(item);
    //   addCardToHistory(
    //     item["subject"],
    //     item["replace-format"],
    //     item["lesson-date"],
    //     item["lesson-time"],
    //     item["group-number"],
    //     item["tg-status"], //tg
    //     item["gs-status"], //gs
    //     i
    //   );
    // });
    // console.log("test");
    ////////////////////////////////////////////////////////////////////////////////////
  });
  // Обработчики щелчка на карточку
  document.addEventListener("click", function (event) {
    // Проверяем, был ли клик на элементе с классом 'edit-button'
    if (event.target.classList.contains("edit-button")) {
      if (currentMode == "edit") {
        saveForm("edit");
      } else if (currentMode == "new") {
        saveForm("new");
      }
      // Получаем id кнопки
      const buttonId = event.target.id;

      let id = parseInt(buttonId, 10);
      currentHistoryID = id;
      currentMode = "edit";
      setVisible("form-container");
      setNewEditVisible("edit");

      saveCard(id); // Сохранение формы в "редактировать"
      loadForm(currentMode); // Загрузка формы в поля ввода
    }
  });

  // Обработчик переключателя "формата"
  buttons.switcher.addEventListener("click", function () {
    switchFormType();
    setNewSubject(lesson_info.subject.innerHTML);
    setDropList();
    setDropListValue();
  });

  // Обработчик на смену урока
  lesson_info.subject.addEventListener("subjectChanged", function (event) {
    setNewSubject(event.detail.value);
  });

  // Обработчик на главную кнопку "отправить/редактировать"
  buttons.send.addEventListener("click", function (event) {
    event.preventDefault();

    buttons.send.classList.add("spin-main");
    let value = buttons.send.innerHTML;
    buttons.send.innerHTML = "";

    sendNewForm();

    const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
    (async () => {
      await sleep(500);
      buttons.send.innerHTML = value;
      buttons.send.classList.remove("spin-main");
    })();
  });

  // Обработчик изменения выпадающего списка формата замен
  lesson_info.replace_format.addEventListener(
    "replace-formatChanged",
    function (event) {
      rememberDropListValues(event.detail.value);
    }
  );

  // обработчки кнопки "Получить историю"
  document
    .getElementById("get-history-button")
    .addEventListener("click", function () {
      let button = document.getElementById("inner-button");
      button.classList.add("spin-history");
      let value = button.innerHTML;
      button.innerHTML = "";

      getHistoryData(true);

      const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
      (async () => {
        await sleep(500);
        button.innerHTML = value;
        button.classList.remove("spin-history");
      })();
    });

  document
    .getElementById("delete-button")
    .addEventListener("click", function () {
      let button = document.getElementById("delete-button");
      button.classList.add("spin-delete");
      let value = button.innerHTML;
      button.innerHTML = "";

      form = history[currentHistoryID - 1];

      let object = {
        initData: initData,
        ID: form["ID"],
      };
      let jsonString = JSON.stringify(object);
      result = sendPostRequest("/postDeleteData", jsonString);
      if (!result.success) {
        showAlert("Произошла ошибка<br>" + result.response, 10, true);
      } else {
        clearForm();
        showAlert("Заявка скоро удалится", 7, false);
      }

      const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
      (async () => {
        await sleep(500);
        button.innerHTML = value;
        button.classList.remove("spin-delete");
      })();
    });
}

/**
 * Отправляет POST запрос по указанному url. В качестве тела запроса отправляет body
 * @param {string} url
 * @param {string} body
 * @returns
 */
function sendPostRequest(url, body) {
  var Request = false;

  if (window.XMLHttpRequest) {
    Request = new XMLHttpRequest();
  } else if (window.ActiveXObject) {
    try {
      Request = new ActiveXObject("Microsoft.XMLHTTP");
    } catch (CatchException) {
      try {
        Request = new ActiveXObject("Msxml2.XMLHTTP");
      } catch (CatchException2) {
        Request = false;
      }
    }
  }

  if (!Request) {
    return { success: false, response: "Невозможно создать XMLHttpRequest" };
  }

  try {
    Request.open("POST", url, false);
    Request.setRequestHeader("Content-Type", "application/json");

    // Преобразуем body в JSON, если это объект
    const jsonBody = typeof body === "object" ? JSON.stringify(body) : body;

    Request.send(jsonBody);

    if (Request.status === 200) {
      return { success: true, response: "" };
    } else {
      let response = Request.responseText;
      if (response === "") {
        response = `${Request.status} (${Request.statusText})`;
      }
      return { success: false, response: response };
    }
  } catch (error) {
    return { success: false, response: error.toString() };
  }
}

// Сохраняет данные из полей ввода, чтобы не потерять их при выборе истории или новой заявки
function saveCard(id) {
  id = id - 1;
  let cardForm = history[id];

  let form = {};
  form["actualFormType"] = cardForm["replace-transfer"];

  if (cardForm[subject] == "НАСТАВНИЧЕСТВО") {
    form["linkIsEnabled"] = false;
    form["isMentoring"] = true;
  } else {
    form["linkIsEnabled"] = true;
    form["isMentoring"] = false;
  }

  if (cardForm["replace-transfer"] == "replace") {
    form["dropListReplacValue"] = cardForm["replace-format"];
    form["dropListTransferValue"] = "Формат";
  } else {
    form["dropListTransferValue"] = cardForm["replace-format"];
    form["dropListReplacValue"] = "Формат";
  }

  form["comment"] = cardForm["comment"];
  form["lesson-date"] = cardForm["lesson-date"];
  form["lesson-time"] = cardForm["lesson-time"];
  form["transfer-time"] = cardForm["transfer-time"];
  form["replace-format"] = cardForm["replace-format"];
  form["subject"] = cardForm["subject"];
  form["group-number"] = cardForm["group-number"];
  form["module"] = cardForm["module"];
  form["lesson"] = cardForm["lesson"];
  form["teacher"] = cardForm["teacher"];
  form["team-leader"] = cardForm["team-leader"];
  form["reason"] = cardForm["reason"];
  form["link"] = cardForm["link"];
  form["imp-info"] = cardForm["imp-info"];
  form["imp-info2"] = cardForm["imp-info2"];
  form["mentoring-inf-1"] = cardForm["mentoring-inf-1"];
  form["mentoring-inf-2"] = cardForm["mentoring-inf-2"];
  form["mentoring-inf-3"] = cardForm["mentoring-inf-3"];

  SAVE["edit"] = form;
}

/**
 * Сохраняет текущую форму
 */
function saveForm(type) {
  let form = {};
  form["actualFormType"] = actualFormType;
  form["linkIsEnabled"] = linkIsEnabled;
  form["isMentoring"] = isMentoring;
  form["dropListReplacValue"] = dropListReplacValue;
  form["dropListTransferValue"] = dropListTransferValue;

  // Проверка на существование элемента 'comment'
  if (lesson_info.comment_input) {
    form["comment"] = lesson_info.comment_input.value;
  } else {
    console.error("Элемент 'comment_input' не найден");
  }
  form["lesson-date"] = lesson_info.date.value;
  form["lesson-time"] = lesson_info.time.value;
  form["transfer-time"] = lesson_info.transfer_time_input.value;
  form["replace-format"] = lesson_info.replace_format.innerHTML;
  form["subject"] = lesson_info.subject.innerHTML;
  form["group-number"] = lesson_info.group_number.value;
  form["module"] = lesson_info.module.value;
  form["lesson"] = lesson_info.lesson.value;
  form["teacher"] = lesson_info.teacher.innerHTML;
  form["team-leader"] = lesson_info.team_leader.innerHTML;
  form["reason"] = lesson_info.reason.value;
  form["link"] = lesson_info.link_input.value;
  form["imp-info"] = lesson_info.imp_info_input.value;
  form["imp-info2"] = lesson_info.imp_info_input_2.value;
  form["mentoring-inf-1"] = lesson_info.mentoring_inf_1.value;
  form["mentoring-inf-2"] = lesson_info.mentoring_inf_2.value;
  form["mentoring-inf-3"] = lesson_info.mentoring_inf_3.value;

  console.log(form["mentoring-inf-1"])
  console.log(form["mentoring-inf-2"])
  console.log(form["mentoring-inf-3"])
  console.log( lesson_info.mentoring_inf_1.value)
  console.log( lesson_info.mentoring_inf_2.value)
  console.log( lesson_info.mentoring_inf_3.value)

  if (type == "new") {
    SAVE["new"] = form;
  } else if (type == "edit") {
    SAVE["edit"] = form;
  }
  console.log("Сохранена форма: ", form);
}

/**
 * Загружает и устанавливает содержимое формы
 */
function loadForm(type) {
  let form;
  if (type == "new") {
    form = SAVE["new"];
  } else if (type == "edit") {
    form = SAVE["edit"];
  }
  console.log("Будет загружена форма: ", form);
  actualFormType = form["actualFormType"];
  linkIsEnabled = form["linkIsEnabled"];
  isMentoring = form["isMentoring"];
  dropListReplacValue = form["dropListReplacValue"];
  dropListTransferValue = form["dropListTransferValue"];

  lesson_info.comment.value = form["comment"];
  lesson_info.date.value = form["lesson-date"];
  lesson_info.time.value = form["lesson-time"];
  lesson_info.transfer_time_input.value = form["transfer-time"];
  lesson_info.replace_format.innerHTML = form["replace-format"];
  lesson_info.subject.innerHTML = form["subject"];
  lesson_info.group_number.value = form["group-number"];
  lesson_info.module.value = form["module"];
  lesson_info.lesson.value = form["lesson"];

  lesson_info.teacher.innerHTML = form["teacher"];

  lesson_info.team_leader.innerHTML = form["team-leader"];
  lesson_info.reason.value = form["reason"];
  lesson_info.link_input.value = form["link"];
  lesson_info.imp_info_input.value = form["imp-info"];
  lesson_info.imp_info_input_2.value = form["imp-info2"];
  lesson_info.mentoring_inf_1.value = form["mentoring-inf-1"];
  lesson_info.mentoring_inf_2.value = form["mentoring-inf-2"];
  lesson_info.mentoring_inf_3.value = form["mentoring-inf-3"];

  console.log(form["mentoring-inf-1"])
  console.log(form["mentoring-inf-2"])
  console.log(form["mentoring-inf-3"])
  console.log( lesson_info.mentoring_inf_1.value)
  console.log( lesson_info.mentoring_inf_2.value)
  console.log( lesson_info.mentoring_inf_3.value)

  switchFormType();
  switchFormType();
  setNewSubject(lesson_info.subject.innerHTML);
  setDropList();
  setDropListValue();
}

function clearForm() {
  lesson_info.comment_input.value = "";
  lesson_info.date.value = "";
  lesson_info.time.value = "";
  lesson_info.transfer_time_input.value = "";
  lesson_info.replace_format.innerHTML = "Формат";
  lesson_info.subject.innerHTML = "Предмет";
  lesson_info.group_number.value = "";
  lesson_info.module.value = "";
  lesson_info.lesson.value = "";
  lesson_info.teacher.innerHTML = "Преподаватель";
  lesson_info.team_leader.innerHTML = "Тимлидер";
  lesson_info.reason.value = "";
  lesson_info.link_input.value = "";
  lesson_info.imp_info_input.value = "";
  lesson_info.mentoring_inf_1.value = "";
  lesson_info.mentoring_inf_2.value = "";
  lesson_info.mentoring_inf_3.value = "";
}

function sendNewForm() {
  let form = {};
  form["replace-transfer"] = actualFormType;

  if (lesson_info.date.value.trim() == "") {
    showWarning(
      lesson_info.date,
      "Пожалуйста, укажите дату урока",
      "top",
      "45%",
      5000
    );
    return;
  }

  // Извлечение года из даты
  const lessonDate = new Date(lesson_info.date.value.trim());
  const year = lessonDate.getFullYear();

  // Проверка года
  if (year < 1900 || year > 9999) {
    showWarning(
      lesson_info.date,
      "Год должен быть в диапазоне от 1900 до 9999",
      "top",
      "45%",
      5000
    );
    return;
  }

  form["lesson-date"] = lesson_info.date.value.trim();

  if (lesson_info.time.value.trim() == "") {
    showWarning(
      lesson_info.time,
      "Пожалуйста, укажите время урока",
      "top",
      "45%",
      5000
    );
    return;
  }
  form["lesson-time"] = lesson_info.time.value.trim();

  if (actualFormType == "transfer") {
    if (lesson_info.transfer_time_input.value.trim() == "") {
      showWarning(
        lesson_info.transfer_time_input,
        "Пожалуйста, укажите время, на которое вы хотите перенести урок",
        "top",
        "45%",
        5000
      );
      return;
    }
    form["transfer-time"] = lesson_info.transfer_time_input.value.trim();

    if (lesson_info.comment_input.value.trim() == "") {
      showWarning(
        lesson_info.comment_input,
        "Пожалуйста, укажите комментарий",
        "top",
        "45%",
        5000
      );
      return;
    }
    form["comment"] = lesson_info.comment_input.value.trim();
  }

  if (lesson_info.replace_format.innerHTML.trim() == "Формат") {
    if (actualFormType == "replace") {
      showWarning(
        lesson_info.replace_format,
        "Пожалуйста, выберите формат замены",
        "top",
        "45%",
        5000
      );
    } else {
      showWarning(
        lesson_info.replace_format,
        "Пожалуйста, выберите формат переноса",
        "top",
        "45%",
        5000
      );
    }

    return;
  }
  form["replace-format"] = lesson_info.replace_format.innerHTML.trim();

  if (lesson_info.teacher.innerHTML.trim() == "Преподаватель") {
    showWarning(
      lesson_info.teacher,
      "Пожалуйста, укажите время ФИО преподавателя",
      "top",
      "45%",
      5000
    );
    return;
  }
  form["teacher"] = lesson_info.teacher.innerHTML.trim();

  if (lesson_info.team_leader.innerHTML.trim() == "Тимлидер") {
    showWarning(
      lesson_info.team_leader,
      "Пожалуйста, укажите тимлидера",
      "top",
      "45%",
      5000
    );
    return;
  }
  form["team-leader"] = lesson_info.team_leader.innerHTML.trim();

  if (lesson_info.subject.innerHTML.trim() == "Предмет") {
    showWarning(
      lesson_info.subject,
      "Пожалуйста, укажите предмет",
      "top",
      "45%",
      5000
    );
    return;
  }
  form["subject"] = lesson_info.subject.innerHTML.trim();

  if (lesson_info.group_number.value.trim() == "") {
    showWarning(
      lesson_info.group_number,
      "Пожалуйста, укажите номер группы",
      "top",
      "45%",
      5000
    );
    return;
  }
  form["group-number"] = lesson_info.group_number.value.trim();

  if (actualFormType != "transfer") {
    if (form["subject"] != "НАСТАВНИЧЕСТВО") {
      // Ссылка
      if (lesson_info.link_input.value.trim() == "") {
        showWarning(
          lesson_info.link_input,
          "Пожалуйста, укажите ссылку на методический материал урока",
          "top",
          "45%",
          5000
        );
        return;
      }

      if (lesson_info.module.value.trim() == "") {
        showWarning(
          lesson_info.module,
          "Пожалуйста, укажите номер модуля урока",
          "top",
          "45%",
          5000
        );
        return;
      }
      form["module"] = lesson_info.module.value.trim();

      if (lesson_info.lesson.value.trim() == "") {
        showWarning(
          lesson_info.lesson,
          "Пожалуйста, укажите номер урока в модуле",
          "top",
          "45%",
          5000
        );
        return;
      }
      form["lesson"] = lesson_info.lesson.value.trim();

      linkIsCorect = linkIsCorrectness(lesson_info.link_input.value.trim());
      if (!linkIsCorect) {
        showAlert(
          "Убедитесь в корректности ссылки\nСылка должна начинаться с https://",
          7,
          true
        );
        return;
      }
      form["link"] = lesson_info.link_input.value.trim();
      // Важная информация
      if (lesson_info.imp_info_input.value.trim() == "") {
        showWarning(
          lesson_info.imp_info_input,
          "Пожалуйста, заполните это поле",
          "top",
          "45%",
          5000
        );
        return;
      }
      form["imp-info"] = lesson_info.imp_info_input.value.trim();
     
      // Важная информация 2
      if (lesson_info.imp_info_input_2.value.trim() == "") {
        showWarning(
          lesson_info.imp_info_input_2,
          "Пожалуйста, заполните это поле",
          "top",
          "45%",
          5000
        );
        return;
      }
      form["imp-info2"] = lesson_info.imp_info_input_2.value.trim();
    } else {
      if (lesson_info.mentoring_inf_1.value.trim() == "") {
        showWarning(
          lesson_info.mentoring_inf_1,
          "Пожалуйста, заполните это поле",
          "top",
          "45%",
          5000
        );
        return;
      }

      if (lesson_info.mentoring_inf_2.value.trim() == "") {
        showWarning(
          lesson_info.mentoring_inf_2,
          "Пожалуйста, заполните это поле",
          "top",
          "45%",
          5000
        );
        return;
      }

      if (lesson_info.mentoring_inf_3.value.trim() == "") {
        showWarning(
          lesson_info.mentoring_inf_3,
          "Пожалуйста, заполните это поле",
          "top",
          "45%",
          5000
        );
        return;
      }
      let mentoring_inf_1 = lesson_info.mentoring_inf_1.value.trim();
      let mentoring_inf_2 = lesson_info.mentoring_inf_2.value.trim();
      let mentoring_inf_3 = lesson_info.mentoring_inf_3.value.trim();
      form["mentoring-inf-1"] = mentoring_inf_1;
      form["mentoring-inf-2"] = mentoring_inf_2;
      form["mentoring-inf-3"] = mentoring_inf_3;
    }
  }

  if (lesson_info.reason.value.trim() == "") {
    showWarning(
      lesson_info.reason,
      "Пожалуйста, укажите причину",
      "top",
      "45%",
      5000
    );
    return;
  }
  form["reason"] = lesson_info.reason.value.trim();

  if (currentMode == "edit") {
    form["ID"] = history[currentHistoryID - 1]["ID"];
    form["google-sheet-status"] =
      history[currentHistoryID - 1]["google-sheet-status"];

    form["emergency-tg-status"] =
      history[currentHistoryID - 1]["emergency-tg-status"];
    form["replace-tg-status"] =
      history[currentHistoryID - 1]["replace-tg-status"];

    form["creation-date"] = history[currentHistoryID - 1]["creation-date"];
    form["creation-time"] = history[currentHistoryID - 1]["creation-time"];
  }

  form["initData"] = initData;

  const jsonString = JSON.stringify(form);
  console.log("Для отправки собраны данные: ", jsonString);

  let result;
  if (currentMode == "new") {
    result = sendPostRequest("/postSetData", jsonString);
  } else if (currentMode == "edit") {
    result = sendPostRequest("/postEditData", jsonString);
  }
  if (!result.success) {
    showAlert("Произошла ошибка<br>" + result.response, 10, true);
  } else {
    clearForm();
    showAlert("", 7, false);
  }
}

/**
 * Отправляет запрос на получении истории и вставляет полученную историю в контейнер истории
 */
function getHistoryData(is_button) {
  // Получаем все элементы с классом "card"
  const cardElements = document.querySelectorAll(".card");

  // Удаляем каждый элемент
  cardElements.forEach((element) => {
    element.remove();
  });

  document.getElementById("inner-button").classList.add("silver-border");

  let initDataString = encodeURIComponent(initData);
  result = sendGetRequest("/getHistoryData", "initData=" + initDataString);
  if (!result.success) {
    if (is_button) {
      showAlert(
        "Не удалось получить историю заявок<br>" + result.response,
        10,
        true
      );
    }

    console.error("Не удалось получить историю заявок" + result.response);
  } else {
    if (is_button) {
      showAlert("История успешно получена" + "", 3, false);
    }
    let object;
    try {
      object = JSON.parse(result.response);
    } catch (error) {
      document.getElementById("inner-button").classList.remove("silver-border");
      return;
    }
    history = object;

    let i = 0;
    history.forEach((item) => {
      i++;
      console.log(item);
      newId = generateRandomId();
      addCardToHistory(
        item["subject"],
        item["replace-format"],
        item["lesson-date"],
        item["lesson-time"],
        item["group-number"],
        item["emergency-tg-status"] || item["replace-tg-status"], //tg
        item["google-sheet-status"], //gs
        newId,
        i
      );
      createCountdownTimer(
        item["creation-date"],
        item["creation-time"],
        item["remaining-time"],
        newId
      );
    });
  }
  document.getElementById("inner-button").classList.remove("silver-border");
}

function generateRandomId() {
  return Math.floor(1000 + Math.random() * 9000).toString();
}

// Функция для создания таймера обратного отсчета
function createCountdownTimer(
  dateString,
  timeString,
  remainingTime,
  timerElementId
) {
  const [minutes, seconds] = remainingTime.split(":").map(Number);
  const targetDate = new Date(`${dateString}T${timeString}`);
  targetDate.setMinutes(targetDate.getMinutes() + minutes);
  targetDate.setSeconds(targetDate.getSeconds() + seconds);

  const timerElement = document.getElementById(timerElementId);
  if (!timerElement) {
    console.error(`Элемент с id "${timerElementId}" не найден`);
    return;
  }

  const timer = setInterval(() => {
    const now = new Date();
    const difference = targetDate - now;

    if (difference > 0) {
      const hours = Math.floor(difference / (1000 * 60 * 60));
      const minutes = Math.floor((difference % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((difference % (1000 * 60)) / 1000);

      timerElement.innerHTML = `${minutes.toString().padStart(2, "0")}:${seconds
        .toString()
        .padStart(2, "0")}`;
    } else {
      clearInterval(timer);
      timerElement.innerHTML = "Время редактирования заявки вышло";
    }
  }, 1000);

  // Возвращаем функцию для остановки таймера
  return () => clearInterval(timer);
}

/**
 * Создает карточку и вставляет ее в контейнер истории
 * @param {string} subject - предмет
 * @param {string} replaceFormat - формат замены
 * @param {string} date - дата урока
 * @param {string} time  - время урока
 * @param {string} groupNumber - номер группы
 * @param {string} id - индекс в массиве
 */
function addCardToHistory(
  subject,
  replaceFormat,
  date,
  time,
  groupNumber,
  tg_status,
  gs_status,
  timer,
  id
) {
  // Создаем HTML-код карточки
  let cardHTML = `
    <div class="card">
      <div class="title">${subject}</div>
      <div class="divider"></div>
      <div class="horizontal">`;

  if (tg_status) {
    cardHTML += `<img class="tg" src="/static/img/tg.svg" alt=""></img>`;
  } else {
    cardHTML += `<img class="tg disabled" src="/static/img/tg.svg" alt=""></img>`;
  }
  // if (gs_status) {
  //   cardHTML += `<img class="gs"  src="/static/img/gs.svg" alt="">`;
  // } else {
  //   cardHTML += `<img class="gs disabled"  src="/static/img/gs.svg" alt="">`;
  // }

  cardHTML += `
      </div>
      <div class="text">${replaceFormat}</div>
      <div class="text">${date} ${time}</div>
      <div class="text">№${groupNumber}</div>
      
      <div class="edit-button" id="${id}">Редактировать</div>
      <div class="text remaining-time">Доступно: <span class="timer text" id="${timer}"></span></div>
    </div>
  `;
  // Находим контейнер истории
  const historyContainer = document.getElementById("history-container");
  // Вставляем новую карточку в конец контейнера
  historyContainer.insertAdjacentHTML("beforeend", cardHTML);
}

/**
 * Устанавливает подписи кнопки и главной страницы.
 * @param {string} type
 */
function setNewEditVisible(type) {
  if (type == "new") {
    document.getElementById("h1-main-title").innerHTML = "НОВАЯ ЗАЯВКА";
    document.getElementById("send-button").innerHTML = "ОТПРАВИТЬ";

    document.getElementById("delete-container").style.display = "none";
  } else if (type == "edit") {
    document.getElementById("h1-main-title").innerHTML = "РЕДАКТИРОВАНИЕ";
    document.getElementById("send-button").innerHTML = "РЕДАКТИРОВАТЬ";

    document.getElementById("delete-container").style.display = "block";
  }
}

// данные для
let ReplacementFormats;
let TransfermentFormats;
/**
 * Отправляет запрос на сервер для получения данных выпадающих списков.
 */
function getSelectData() {
  let initDataString = encodeURIComponent(initData);
  result = sendGetRequest("/getData", "initData=" + initDataString);
  if (!result.success) {
    showAlert(
      "Не удалось получить данные выпадающих списков<br>Обновите страницу<br>" +
        result.response,
      10,
      true
    );
    ReplacementFormats = ["Формат 1", "Формат 2", "Формат 3"];
    TransfermentFormats = ["Формат 1", "Формат 2", "Формат 3"];
  } else {
    console.log("Полученны данные выпадающих списков: ", result.response);
    const SelectData = JSON.parse(result.response);
    updateListContent("drop-list-subject", SelectData.Objects);
    updateListContent("drop-list-teacher", SelectData.Teachers);
    updateListContent("drop-list-teamleader", SelectData.TeamLeaders);

    ReplacementFormats = SelectData.ReplacementFormats;
    TransfermentFormats = SelectData.TransfermentFormats;
    setDropList();
  }
}

/**
 * Отправляет GET запрос по указанному url. В качестве параметров отправляет urlQuery.
 * Нербходимо самостоятельно заранее закодировать urlQuery. Указать надо без "?"
 * @param {string} url
 * @param {string} urlQuery
 * @returns
 */
function sendGetRequest(url, urlQuery) {
  var Request = false;

  if (window.XMLHttpRequest) {
    Request = new XMLHttpRequest();
  } else if (window.ActiveXObject) {
    try {
      Request = new ActiveXObject("Microsoft.XMLHTTP");
    } catch (CatchException) {
      try {
        Request = new ActiveXObject("Msxml2.XMLHTTP");
      } catch (CatchException2) {
        Request = false;
      }
    }
  }

  if (!Request) {
    return { success: false, response: "Невозможно созд��ть XMLHttpRequest" };
  }

  try {
    // Изменяем эту часть, чтобы избежать двойного кодирования
    const fullUrl = urlQuery ? `${url}?${urlQuery}` : url;

    console.log("fullUrl    ", fullUrl);
    Request.open("GET", fullUrl, false);
    Request.send();

    if (Request.status === 200) {
      return { success: true, response: Request.responseText };
    } else {
      let response;
      switch (Request.status) {
        case 404:
          response = "404 (Not Found)";
          break;
        case 403:
          response = "403 (Forbidden)";
          break;
        case 500:
          response = "500 (Internal Server Error)";
          break;
        default:
          response = `${Request.status} (${
            Request.statusText || "Unknown Error"
          })`;
      }
      return { success: false, response: response };
    }
  } catch (error) {
    return { success: false, response: "Network Error" };
  }
}

/**
 * Устанавливает содержимое выпадающего списка форматов замен в зависимости от текущего actualFormType
 */
function setDropList() {
  if (actualFormType == "transfer") {
    updateListContent("drop-list-format", TransfermentFormats);
    console.log("Обновлено содержимое списка на: ", TransfermentFormats);
  } else if (actualFormType == "replace") {
    updateListContent("drop-list-format", ReplacementFormats);
    console.log("Обновлено содержимое списка на: ", ReplacementFormats);
  }
}

/**
 * Запоминает выбор формата замен для "замены" или "переноса" в зависимости от actualFormType
 * @param {string} replaceFormat
 */
function rememberDropListValues(replaceFormat) {
  if (actualFormType == "transfer") {
    dropListTransferValue = replaceFormat;
  } else if (actualFormType == "replace") {
    dropListReplacValue = replaceFormat;
  }
}

/**
 * Устанавливает выбор пользователя выпадающего списка форматов замен в зависимости от текущего actualFormType
 */
function setDropListValue() {
  if (actualFormType == "transfer") {
    lesson_info.replace_format.innerHTML = dropListTransferValue;
  } else if (actualFormType == "replace") {
    lesson_info.replace_format.innerHTML = dropListReplacValue;
  }
}

/**
 * Вставляет в ul с id = elementId список li.
 * @param {string} elementId
 * @param {[]} items
 * @returns
 */
function updateListContent(elementId, items) {
  // Получаем элемент по ID
  const element = document.getElementById(elementId);

  // Проверяем, существует ли элемент
  if (!element) {
    console.error(`Элемент с ID "${elementId}" не найден`);
    return;
  }

  // Очищаем текущее содержимое элемента
  element.innerHTML = "";

  // Создаем новый список
  const ul = document.createElement("ul");

  // Добавляем каждый элемент массива как <li> в список
  items.forEach((item) => {
    const li = document.createElement("li");
    li.textContent = item; // Используем textContent для безопасности
    ul.appendChild(li);
  });

  // Добавляем созданный список в элемент
  element.appendChild(ul);
}

/**
 * Меняет оформление формы в зависимости от выбранного урока.
 * @param {string} subject
 */
function setNewSubject(subject) {
  console.log("Предмет изменен на:", subject);
  if (subject == "НАСТАВНИЧЕСТВО") {
    setLinkType(false);
    setImpInfoMentorType(true);
    isMentoring = true;
  } else {
    setLinkType(true);
    setImpInfoMentorType(false);
  }
}

/**
 * Переключает необходимость важной информации, скрывает-показывает контейнер.
 * @param {bool} isEnabled
 */
function setImpInfoMentorType(isEnabled) {
  isMentoring = isEnabled;
  if (isMentoring) {
    if (actualFormType != "transfer") {
      lesson_info.imp_info_container.style.display = "none";
      lesson_info.imp_info_mentor_container.style.display = "flex";
    }
  } else {
    if (actualFormType != "transfer") {
      lesson_info.imp_info_container.style.display = "flex";
      lesson_info.imp_info_mentor_container.style.display = "none";
    }
  }
}

/**
 * Переключает необходимость ссылки, меняет визуальное оформление ссылки.
 * @param {bool} isEnabled
 */
function setLinkType(isEnabled) {
  linkIsEnabled = isEnabled;
  if (isEnabled) {
    // ссылка нужна
    console.log("ссылка нужна");
    lesson_info.lint_title.innerHTML =
      "Ссылка на методический материал по уроку*";

    lesson_info.lint_title.classList.remove("disabled");
    lesson_info.link_input.classList.remove("disabled-border");
    lesson_info.link_input.classList.remove("disabled");

    lesson_info.lesson.classList.remove("disabled");
    lesson_info.lesson.classList.remove("disabled-border");
    document.getElementById("lesson-title").innerHTML = "Урок*";

    lesson_info.module.classList.remove("disabled");
    lesson_info.module.classList.remove("disabled-border");
    document.getElementById("module-title").innerHTML = "Модуль*";

    modifyClassForElements("lesson-number", "disabled", false);
  } else {
    // Ссылка не нужна
    console.log("Ссылка не нужна");
    lesson_info.lint_title.innerHTML =
      "Ссылка на методический материал по уроку. При наставничестве не обязательна.";
    lesson_info.lint_title.classList.add("disabled");
    lesson_info.link_input.classList.add("disabled-border");
    lesson_info.link_input.classList.add("disabled");

    lesson_info.lesson.classList.add("disabled");
    lesson_info.lesson.classList.add("disabled-border");
    document.getElementById("lesson-title").innerHTML = "Урок";

    lesson_info.module.classList.add("disabled");
    lesson_info.module.classList.add("disabled-border");
    document.getElementById("module-title").innerHTML = "Модуль";

    modifyClassForElements("lesson-number", "disabled", true);
  }
}

function modifyClassForElements(targetClass, newClass, addOrRemove = true) {
  const elements = document.getElementsByClassName(targetClass);

  for (let element of elements) {
    if (addOrRemove) {
      // Добавляем новый класс
      element.classList.add(newClass);
    } else {
      // Удаляем класс
      element.classList.remove(newClass);
    }
  }
}

// Примеры использования:
// Добавить тег:
// modifyTagForClass('highlight', 'strong', true);
// Удалить тег:
// modifyTagForClass('highlight', 'strong', false);

/**
 * Переключатель режимов замена-перенос
 */
function switchFormType() {
  if (actualFormType == "replace") {
    setTransferForm();
  } else if (actualFormType == "transfer") {
    setReplaceForm();
  }
}

/**
 * Добавляет, удаляет черточки под кнопками, показывает или скрывает контейнеры.
 */
function setReplaceForm() {
  console.log("Выбран тип ЗАМЕНА");
  actualFormType = "replace";

  buttons.transfer_button_style.classList.remove("selected");
  buttons.replace_button_style.classList.add("selected");

  lesson_info.reason_title.innerHTML = "Причина замены*";

  lesson_info.comment_container.style.display = "none";

  lesson_info.transfer_time_container.style.display = "none";
  lesson_info.link_container.style.display = "flex";
  lesson_info.imp_info_container.style.display = "flex";
  lesson_info.imp_info_mentor_container.style.display = "flex";
}

/**
 * Добавляет, удаляет черточки под кнопками, показывает или скрывает контейнеры.
 */
function setTransferForm() {
  console.log("Выбран тип ПЕРЕНОС");
  actualFormType = "transfer";

  buttons.replace_button_style.classList.remove("selected");
  buttons.transfer_button_style.classList.add("selected");

  lesson_info.reason_title.innerHTML = "Причина переноса*";

  lesson_info.comment_container.style.display = "flex";

  lesson_info.transfer_time_container.style.display = "flex";
  lesson_info.link_container.style.display = "none";
  lesson_info.imp_info_container.style.display = "none";
  lesson_info.imp_info_mentor_container.style.display = "none";
}

/**
 * setVisible переключает вид главной страницы
 * @param {string} type - "form-container"/"history-container"
 */
function setVisible(type) {
  if (type == "form-container") {
    main_sections.history_container.style.display = "none";
    main_sections.form_container.style.display = "inline-block";
    start_visible = "form-container";
  } else if (type == "history-container") {
    main_sections.history_container.style.display = "flex";
    main_sections.form_container.style.display = "none";
    start_visible = "history-container";
  }
}

/**
 * Функция showAlert отображает уведомление для пользователя с заданным текстом и определённым стилем.
 *
 * @param {string} error_text - Текст уведомления, который будет показан пользователю.
 * @param {number} durationSeconds - Продолжительность анимации (в секундах), после которой уведомление исчезнет.
 * @param {boolean} isError - Флаг, указывающий на тип уведомления. Если true, уведомление будет красным и помечено как ошибка. Если false, уведомление будет зелёным и помечено как успешно.
 */
function showAlert(error_text, durationSeconds, isError) {
  function easeInOutQuad(t) {
    return t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
  }

  // Устанавливаем стили и текст уведомления в зависимости от т��па сообщения
  if (isError) {
    error_alert.alert_close_button.style.color = "#ff0000"; // Красный цвет для ошибок
    error_alert.title.innerHTML = "ОШИБКА";
  } else {
    error_alert.title.innerHTML = "УСПЕШНО";
    error_alert.alert_close_button.style.color = "#1ede00"; // Зелёный цвет для успешных сообщений
  }

  // Устанавливаем текст уведомления
  error_alert.error_description.innerHTML = error_text;

  // Добавляем обработчик клика для кнопки закрытия уведомления
  error_alert.alert_close_button.addEventListener("click", function () {
    moveElement(error_alert.alert_container, "toRight"); // Перемещаем уведомление за пределы экрана при закрытии
  });

  // Перемещаем уведомление в центр экрана
  moveElement(error_alert.alert_container, "toCenter");

  // Запускаем анимацию ширины уведомления и перемещаем его за пределы экрана по окончании анимации
  animateWidth(
    error_alert.loading,
    durationSeconds,
    easeInOutQuad,
    function () {
      moveElement(error_alert.alert_container, "toRight");
    }
  );
}

/**
 *  animateWidth активирует плавное увеличение ширины обьекта с 0 до 100%
 */
function animateWidth(
  element,
  durationSeconds,
  easingFunction = (t) => t,
  callback
) {
  if (!(element instanceof Element)) {
    console.error("Переданный аргумент не является DOM элементом");
    return;
  }

  const fps = 60;
  const totalFrames = durationSeconds * fps;

  function step(timestamp) {
    if (!step.startTime) step.startTime = timestamp;
    const elapsed = timestamp - step.startTime;
    const progress = Math.min(elapsed / (durationSeconds * 1000), 1);

    const easedProgress = easingFunction(progress);
    const currentWidth = easedProgress * 90;

    element.style.width = currentWidth + "%";

    if (progress < 1) {
      requestAnimationFrame(step);
    } else {
      // Анимация завершена, вызываем callback, если он предоставлен
      if (typeof callback === "function") {
        callback();
      }
    }
  }

  requestAnimationFrame(step);
}

/**
 * Функция moveElement управляет позиционированием HTML-элемента, изменяя его CSS-классы в зависимости от указанного направления.
 *
 * @param {HTMLElement} element - HTML-элемент, который требуется переместить.
 * @param {string} direction - Направление перемещения элемента. Может принимать значения:
 *   - "toCenter": Перемещает элемент в центр экрана, добавляя класс "center-screen" и убирая класс "off-screen".
 *   - "toRight": Перемещает элемент в правую часть экрана, добавляя класс "off-screen" и убирая класс "center-screen".
 */
function moveElement(element, direction) {
  if (direction === "toCenter") {
    element.classList.remove("off-screen");
    element.classList.add("center-screen");
  } else if (direction === "toRight") {
    element.classList.remove("center-screen");
    element.classList.add("off-screen");
  }
}

/**
 * Преобразует все строковые значения в объекте (и его вложенных объектах) в соответствующие HTML элементы.
 * Если элемент с указанным id не найден, выводится предупреждение в консоль.
 *
 * @param {Object} objects - Объект, содержащий другие объекты, значения которых нужно преобразовать
 */
function convertIdsToElements(objects) {
  /**
   * Рекурсивно обрабатывает объект, заменяя строковые значения на HTML элементы
   *
   * @param {Object} obj - Объект для обработки
   */
  function processObject(obj) {
    for (let key in obj) {
      if (typeof obj[key] === "string") {
        const id = obj[key]; // Используем значение как id
        const element = document.getElementById(id);
        if (element) {
          console.log(`Элемент с id "${id}" найден для ключа "${key}"`);
          obj[key] = element; // Заменяем строку на HTML элемент
        } else {
          console.warn(`Элемент с id "${id}" не найден для ключа "${key}"`);
        }
      } else if (typeof obj[key] === "object" && obj[key] !== null) {
        console.log(`Обработка вложенного объекта для ключа "${key}"`);
        processObject(obj[key]); // Рекурсивный вызов для вложенных объектов
      }
    }
  }

  // Обрабатываем каждый объект в переданном objects
  for (let objName in objects) {
    console.log(`Начало обработки объекта "${objName}"`);
    processObject(objects[objName]);
  }
}

/**
 * setTg инициализирует Telegram Web App и возвращает объекты tg и initData
 * @param {number} [maxAttempts=3] Максимальное количество попыток инициализации
 * @param {number} [delay=1000] Задержка между попытками в миллисекундах
 * @returns {Object|null} Объект с свойствами tg и initData или null в случае неудачи
 */
function initializeTg(maxAttempts = 3, delay = 1000) {
  function sleep(ms) {
    const start = Date.now();
    while (Date.now() - start < ms) {}
  }

  for (let attempts = 1; attempts <= maxAttempts; attempts++) {
    if (window.Telegram && window.Telegram.WebApp) {
      const tg = window.Telegram.WebApp;
      const initData = tg.initData;

      // Проверяем, что initData не пустой и не равен ""
      if (!initData || initData === "") {
        console.warn(
          `Попытка ${attempts}: initData пуст или равен "". Возможно, приложение запущено не в Telegram.`
        );
      } else {
        // Вызываем tg.ready() для сообщения Telegram, что приложение готово
        tg.ready();

        // Добавим проверку на поддержку основных методов
        if (
          typeof tg.sendData !== "function" ||
          typeof tg.expand !== "function"
        ) {
          console.warn(
            `Попытка ${attempts}: Некоторые ожидаемые методы Telegram Web App отсутствуют.`
          );
        } else {
          console.log(
            `Telegram Web App успешно инициализирован (попытка ${attempts})`
          );
          return { tg, initData };
        }
      }
    } else {
      console.error(
        `Попытка ${attempts}: Telegram Web App не найден. Проверьте подключение библиотеки telegram-web-app.js.`
      );
    }

    if (attempts < maxAttempts) {
      console.log(`Повторная попытка через ${delay}мс...`);
      sleep(delay);
    }
  }

  console.error(
    `Не удалось инициализировать Telegram Web App после ${maxAttempts} попыток.`
  );
  return null;
}

/**
 * setTgBackButtonHandler назначает действие кнопки "назад" в Telegram Web App
 * и отображает эту кнопку в интерфейсе приложения.
 */
function setTgBackButtonHandler() {
  // Назначаем обработчик события нажатия кнопки "назад"
  tg.onEvent("backButtonClicked", function () {
    if (currentMode == "history") {
      // Если переход осуществлен из состояния "заполнения формы", то сохраняем форму
      if (currentMode == "edit") {
        saveForm("edit");
      } else if (currentMode == "new") {
        saveForm("new");
      }

      setVisible("form-container");
      buttons.new_form.classList.add("white-line");
      buttons.history.classList.remove("white-line");

      currentMode = "new";
      setNewEditVisible("new"); // Изменение текста кнопок
      loadForm(currentMode);
    } else if (currentMode == "edit") {
      // Если переход осуществлен из состояния "заполнения формы", то сохраняем форму
      if (currentMode == "edit") {
        saveForm("edit");
      } else if (currentMode == "new") {
        saveForm("new");
      }

      setVisible("history-container");
      buttons.history.classList.add("white-line");
      buttons.new_form.classList.remove("white-line");

      getHistoryData(false);
      currentMode = "history";
      setNewEditVisible("new"); // Изменение текста кнопок
    } else if (currentMode == "new") {
      window.Telegram.WebApp.close();
    }
  });

  // Отображаем кнопку "Назад" в интерфейсе Telegram Web App
  tg.BackButton.show();
}

/**
 * Проверяет корректность ссылки на основе белого и черного списков.
 *
 * @param {string} link - Проверяемая ссылка.
 * @returns {boolean} - true, если ссылка корректна, false в противном случае.
 */
function linkIsCorrectness(link) {
  // Белый список разрешенных префиксов
  const whiteList = ["https://"];

  // Черный список запрещенных префиксов
  const blackList = [];

  // Проверка на наличие ссылки в черном списке
  if (blackList.some((prefix) => link.startsWith(prefix))) {
    return false;
  }

  // Проверка на наличие ссылки в белом списке
  return whiteList.some((prefix) => link.startsWith(prefix));
}

// Глобальные переменные для отслеживания текущего предупреждения
let currentWarning = null; // Хранит текущий элемент предупреждения
let currentWarningTimeout = null; // Хранит таймер для автоматического скрытия предупреждения

/**
 * Показывает предупреждение рядом с целевым элементом.
 * @param {Element} targetElement - Элемент, рядом с которым нужно показать предупреждение.
 * @param {string} message - Текст предупреждения.
 * @param {string} position - Позиция предупреждения относительно целевого элемента (по умолчанию "top").
 * @param {string} percent - Горизонтальное смещение стрелочки предупреждения в процентах (по умолчанию "50%").
 * @param {number} duration - Продолжительность показа предупреждения в миллисекундах (по умолчанию 3000).
 */
function showWarning(
  targetElement,
  message,
  position = "top",
  percent = "50%",
  duration = 3000
) {
  console.log("showWarning called", {
    targetElement,
    message,
    position,
    percent,
    duration,
  });

  // Удаляем предыдущее предупреждение, если оно существует
  if (currentWarning) {
    currentWarning.remove();
  }

  // Очищаем предыдущий таймер, если он существует
  if (currentWarningTimeout) {
    clearTimeout(currentWarningTimeout);
  }

  // Проверяем, является ли targetElement допустимым DOM-элементом
  if (!targetElement || !(targetElement instanceof Element)) {
    console.error("Invalid targetElement");
    return;
  }

  // Создаем уникальный идентификатор для предупреждения
  const warningId = `warning-${
    targetElement.id || Math.random().toString(36).substr(2, 9)
  }`;

  // Создаем элемент предупреждения
  const warning = document.createElement("div");
  warning.id = warningId;
  warning.className = `warning warning-${position}`;
  warning.textContent = message;
  warning.style.visibility = "hidden";
  warning.style.opacity = "0";
  document.body.appendChild(warning);

  currentWarning = warning;

  console.log("Warning element created", warning);

  // Устанавливаем CSS-переменную для позиционирования псевдоэлемента
  document.documentElement.style.setProperty("--pseudo-left", percent);

  // Используем setTimeout, чтобы дать браузеру время на отрисовку предупреждения
  setTimeout(() => {
    // Полу��аем позицию целевого элемента
    const targetRect = targetElement.getBoundingClientRect();
    const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
    const scrollLeft =
      window.pageXOffset || document.documentElement.scrollLeft;

    // Позиционируем предупреждение
    warning.style.left = `${
      targetRect.left + scrollLeft + targetRect.width / 2
    }px`;
    warning.style.top = `${targetRect.top + scrollTop - 10}px`;
    warning.style.transform = "translate(-50%, -100%)";
    warning.style.visibility = "visible";

    console.log("Warning styles set", {
      position: warning.style.position,
      top: warning.style.top,
      left: warning.style.left,
      transform: warning.style.transform,
      zIndex: warning.style.zIndex,
      pseudoLeft: percent,
    });

    // Анимируем появление предупреждения
    requestAnimationFrame(() => {
      warning.style.opacity = "1";
      targetElement.scrollIntoView({ behavior: "smooth", block: "center" });
    });

    console.log("Warning position:", warning.getBoundingClientRect());
    console.log("Target position:", targetRect);

    // Устанавливаем таймер для автоматического скрытия предупреждения
    if (duration !== Infinity) {
      currentWarningTimeout = setTimeout(() => {
        hideWarning(targetElement);
      }, duration);
    }
  }, 0);
}

/**
 * Скрывает текущее предупреждение.
 * @param {Element} targetElement - Элемент, рядом с которым было показано предупреждение.
 */
function hideWarning(targetElement) {
  if (currentWarning) {
    console.log("Hiding warning", currentWarning);
    // Анимируем исчезновение предупреждения
    currentWarning.style.opacity = "0";
    setTimeout(() => {
      currentWarning.remove();
      currentWarning = null;
      console.log("Warning removed");
    }, 300);
  } else {
    console.log("No active warning to hide");
  }

  // Сбрасываем CSS-переменную для позиционирования псевдоэлемента
  document.documentElement.style.removeProperty("--pseudo-left");
}

document.addEventListener("touchend", function (event) {
  var target = event.target;
  var isInput = target.tagName === "INPUT" || target.tagName === "TEXTAREA";

  if (!isInput) {
    // Найти все input и textarea элементы
    var inputs = document.querySelectorAll("input, textarea");

    // Убрать фокус со всех полей ввода
    inputs.forEach(function (input) {
      input.blur();
    });
  }
});
