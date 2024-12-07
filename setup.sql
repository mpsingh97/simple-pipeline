--
-- PostgreSQL database dump
--

-- Dumped from database version 14.15
-- Dumped by pg_dump version 14.15 (Ubuntu 14.15-1.pgdg20.04+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: tasks; Type: TABLE; Schema: public; Owner: user
--

CREATE TABLE public.tasks (
    task_id character varying(40) NOT NULL,
    ingest_status character varying(10),
    transcode_status character varying(10),
    metadata_gen_status character varying(10),
    assemble_status character varying(10),
    publish_status character varying(10),
    overridden_by character varying(40),
    host_machine character varying(48),
    process_id integer,
    retries integer,
    start_time timestamp without time zone,
    end_time timestamp without time zone,
    error_message text
);


ALTER TABLE public.tasks OWNER TO "user";

--
-- Data for Name: tasks; Type: TABLE DATA; Schema: public; Owner: user
--

COPY public.tasks (task_id, ingest_status, transcode_status, metadata_gen_status, assemble_status, publish_status, overridden_by, host_machine, process_id, retries, start_time, end_time, error_message) FROM stdin;
\.


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: user
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (task_id);


--
-- PostgreSQL database dump complete
--

