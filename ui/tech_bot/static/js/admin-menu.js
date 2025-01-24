document.addEventListener("DOMContentLoaded", function () {
  document
    .getElementById("updateSelectButton")
    .addEventListener("click", sendUpdateSelectButton);
});

function initialize() {
  token = document.getElementById("token");

  updateTeachersButton = document.getElementById("updateTeachersButton");
  consoleText = document.getElementById("consoleText");
}

let UpdateSelectButtonIsClicled = false;
function sendUpdateSelectButton() {
  if (UpdateSelectButtonIsClicled) {
    return
  }
  UpdateSelectButtonIsClicled = true
  updateSelectButton = document.getElementById("updateSelectButton");
  updateSelectButton.classList.add("disabled");

  token = document.getElementById("token").value;

  const urlQuery =
    "token=" + encodeURIComponent(token) + "&command=updateSelectData";
  result = sendGetRequest("/internal/admin", urlQuery);
  console.log(result);

  consoleText = document.getElementById("consoleText");

  consoleText.innerHTML = result.response + "<br>" + consoleText.innerHTML;

  console.log(consoleText.innerHTML);
  updateSelectButton.classList.remove("disabled");
  UpdateSelectButtonIsClicled = false
}

/**
 * Отправляет GET запрос по указанному url. В качестве параметров отправляет urlQuery.
 * Необходимо самостоятельно заранее закодировать urlQuery. Указать надо без "?"
 * @param {string} url
 * @param {string} urlQuery
 * @returns {{success: boolean, response: string, statusCode: number, statusText: string, responseBody: string}}
 */
function sendGetRequest(url, urlQuery) {
  const Request = new XMLHttpRequest();

  if (!Request) {
    return {
      success: false,
      response: "Невозможно создать XMLHttpRequest",
      statusCode: 0,
      statusText: "Error",
      responseBody: ""
    };
  }

  const fullUrl = urlQuery ? `${url}?${urlQuery}` : url;

  console.log("fullUrl:", fullUrl);

  Request.open("GET", fullUrl, false);  // false для синхронного запроса

  try {
    Request.send();

    const response = {
      success: Request.status === 200,
      statusCode: Request.status,
      statusText: Request.statusText || "Unknown Status",
      responseBody: Request.responseText,
    };

    if (response.success) {
      response.response = Request.responseText;
    } else {
      response.response = `${response.statusCode} (${response.statusText}): ${response.responseBody}`;
    }

    return response;
  } catch (error) {
    return {
      success: false,
      response: "Network Error: " + error.message,
      statusCode: 0,
      statusText: "Error",
      responseBody: ""
    };
  }
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
