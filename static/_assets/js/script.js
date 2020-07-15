$('button[type="submit"]').on("click", function() {
  setTimeout(function() {
    $(this).prop("disabled", true);
  });
});
$(function () {
  $('[data-toggle="tooltip"]').tooltip()
});

$("img").on("error", function () {
  $(this).attr("src", "_assets/img/logo-1x-no-blur.png");
});
