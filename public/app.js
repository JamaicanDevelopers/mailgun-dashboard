var windowOptions = 'menubar=no,toolbar=no,location=no,directories=no,status=no,resizable=yes,scrollbars=yes'

$('.show-body').on('click', function(event) {
    event.preventDefault();
    window.open($(this).attr('href'), $(this).data('name'), windowOptions);
});

$('.resend-message').on('click', function(event) {
    event.preventDefault();
    var to = window.prompt("Resend this message to?", $(this).data('to'));

    if (null == to) {
        return;
    }

    $.post($(this).attr('href'), {key: $(this).data('key'), to: to}, function(data) {
        window.alert("Message resent.");
    });
})
