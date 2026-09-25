FROM postgres:16-alpine
COPY entrypoint.sh /entrypoint.sh
COPY sql /sql
RUN chmod +x /entrypoint.sh
CMD ["/entrypoint.sh"]
