window.addEventListener("DOMContentLoaded", (event) => {
  document.body.addEventListener("showMessage", function(evt){
    alert(evt.detail.value);
  })
});

