// Переменные для отслеживания свайпа
let startX;
let startY;
let distX;
let distY;
let threshold = 100; // минимальное расстояние для свайпа
let restraint = 100; // максимальное отклонение по другой оси
let allowedTime = 300; // максимальное время свайпа
let startTime;

document.addEventListener(
  "touchstart",
  function (e) {
    let touchobj = e.changedTouches[0];
    startX = touchobj.pageX;
    startY = touchobj.pageY;
    startTime = new Date().getTime(); // время начала касания
  },
  false
);

document.addEventListener(
  "touchend",
  function (e) {
    let touchobj = e.changedTouches[0];
    distX = touchobj.pageX - startX; // расстояние по горизонтали
    distY = touchobj.pageY - startY; // расстояние по вертикали
    let elapsedTime = new Date().getTime() - startTime; // время свайпа

    // Проверяем, был ли свайп достаточно быстрым и длинным
    if (elapsedTime <= allowedTime) {
      if (Math.abs(distX) >= threshold && Math.abs(distY) <= restraint) {
        // Горизонтальный свайп
        if (distX > 0) {
          //////////////////////////////////////////
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
          //////////////////////////////////////////
        } else {
          //////////////////////////////////////////
          if (currentMode == "edit") {
            saveForm("edit");
          } else if (currentMode == "new") {
            saveForm("new");
          }

          setVisible("history-container");
          buttons.history.classList.add("white-line");
          buttons.new_form.classList.remove("white-line");

          getHistoryData();
          currentMode = "history";
          setNewEditVisible("new"); // Изменение текста кнопок
          //////////////////////////////////////////
        }
      } else if (Math.abs(distY) >= threshold && Math.abs(distX) <= restraint) {
        // Вертикальный свайп
        if (distY > 0) {
          console.log("Свайп вниз");
          // Здесь можно запустить нужную функцию для свайпа вниз
        } else {
          console.log("Свайп вверх");
          // Здесь можно запустить нужную функцию для свайпа вверх
        }
      }
    }
  },
  false
);
