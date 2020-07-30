if (!window.sgtm.dev_mode) {
  Sentry.init({
    dsn: 'https://e61830c1fa57411b9b2ce72e4edb47cc@o419562.ingest.sentry.io/5371550',
    release: window.sgtm.release.version,
  });
}
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

$('body.page-open').ready(function() {
  var chart = document.getElementById('myChart');
  if (chart == null) { return; }
  var ctx = chart.getContext('2d');
  var myChart = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'],
      datasets: [{
        label: 'Uploads by day of week',
        data: window.sgtm.open.uploadsbyweekday,
        backgroundColor: [
          'rgba(255, 99, 132, 0.2)',
          'rgba(54, 162, 235, 0.2)',
          'rgba(255, 206, 86, 0.2)',
          'rgba(75, 192, 192, 0.2)',
          'rgba(153, 102, 255, 0.2)',
          'rgba(255, 159, 64, 0.2)',
          'rgba(159, 255, 64, 0.2)'
        ],
        borderColor: [
          'rgba(255, 99, 132, 1)',
          'rgba(54, 162, 235, 1)',
          'rgba(255, 206, 86, 1)',
          'rgba(75, 192, 192, 1)',
          'rgba(153, 102, 255, 1)',
          'rgba(255, 159, 64, 1)',
          'rgba(159, 255, 64, 1)'
        ],
        borderWidth: 1
      }]
    },
    options: {
      scales: {
        yAxes: [{
          ticks: {
            beginAtZero: true
          }
        }]
      }
    }
  });
});
