$('button[type="submit"]').on("click", function() {
  $(this).prop("disabled", true);
});
$(function () {
  $('[data-toggle="tooltip"]').tooltip()
});
