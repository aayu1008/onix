/*
    Onix CMDB - Copyright (c) 2018-2019 by www.gatblau.org

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
    Unless required by applicable law or agreed to in writing, software distributed under
    the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
    either express or implied.
    See the License for the specific language governing permissions and limitations under the License.

    Contributors to this project, hereby assign copyright in this code to the project,
    to be licensed under the same terms as the rest of the code.

*/
DO
$$
BEGIN

/*
  gets an item by its natural key.
  use: select * from item('the_item_key')
 */
CREATE OR REPLACE FUNCTION item(key_param character varying)
  RETURNS TABLE(
    id bigint,
    key character varying,
    name character varying,
    description text,
    status smallint,
    item_type_id integer,
    meta jsonb,
    tag text[],
    attribute hstore,
    version bigint,
    created timestamp(6) with time zone,
    updated timestamp(6) with time zone,
    changedby character varying,
    transaction_ref uuid
  )
  LANGUAGE 'plpgsql'
  COST 100
  STABLE
AS $BODY$
BEGIN
  RETURN QUERY SELECT
     i.id,
     i.key,
     i.name,
     i.description,
     i.status,
     i.item_type_id,
     i.meta,
     i.tag,
     i.attribute,
     i.version,
     i.created,
     i.updated,
     i.changedby,
     i.transaction_ref
   FROM item i
   WHERE i.key = key_param;
END;
$BODY$;

ALTER FUNCTION item(character varying)
  OWNER TO onix;

/*
  gets an item_type by its natural key.
  use: select * from item_type('the_item_type_key')
 */
CREATE OR REPLACE FUNCTION item_type(key_param character varying)
  RETURNS TABLE(
    id integer,
    key character varying,
    name character varying,
    description text,
    attr_valid hstore,
    system boolean,
    version bigint,
    created timestamp(6) with time zone,
    updated timestamp(6) with time zone,
    changedby character varying
  )
  LANGUAGE 'plpgsql'
  COST 100
  STABLE
AS $BODY$
BEGIN
  RETURN QUERY SELECT
    i.id,
    i.key,
    i.name,
    i.description,
    i.attr_valid,
    i.system,
    i.version,
    i.created,
    i.updated,
    i.changedby
  FROM item_type i
  WHERE i.key = key_param;
END;
$BODY$;

ALTER FUNCTION item_type(character varying)
  OWNER TO onix;

/*
  gets a link by its natural key.
  use: select * from link('the_link_key')
 */
CREATE OR REPLACE FUNCTION link(key_param character varying)
  RETURNS TABLE(
    id bigint,
    key CHARACTER VARYING(200),
    link_type_id integer,
    start_item_id bigint,
    end_item_id bigint,
    description text,
    meta jsonb,
    tag text[],
    attribute hstore,
    version bigint,
    created TIMESTAMP(6) WITH TIME ZONE,
    updated timestamp(6) WITH TIME ZONE,
    changedby CHARACTER VARYING(100),
    transaction_ref UUID
  )
  LANGUAGE 'plpgsql'
  COST 100
  STABLE
AS $BODY$
BEGIN
  RETURN QUERY SELECT
     l.id,
     l.key,
     l.link_type_id,
     l.start_item_id,
     l.end_item_id,
     l.description,
     l.meta,
     l.tag,
     l.attribute,
     l.version,
     l.created,
     l.updated,
     l.changedby,
     l.transaction_ref
  FROM link l
  WHERE l.key = key_param;
END;
$BODY$;

ALTER FUNCTION link(character varying)
  OWNER TO onix;

/*
  gets a link_type by its natural key.
  use: select * from link_type('the_link_type_key')
 */
CREATE OR REPLACE FUNCTION link_type(key_param character varying)
  RETURNS TABLE(
    id integer,
    key character varying,
    name character varying,
    description text,
    attr_valid hstore,
    system boolean,
    version bigint,
    created timestamp(6) with time zone,
    updated timestamp(6) with time zone,
    changedby character varying
  )
  LANGUAGE 'plpgsql'
  COST 100
  STABLE
AS $BODY$
BEGIN
  RETURN QUERY SELECT
     l.id,
     l.key,
     l.name,
     l.description,
     l.attr_valid,
     l.system,
     l.version,
     l.created,
     l.updated,
     l.changedby
   FROM link_type l
   WHERE l.key = key_param;
END;
$BODY$;

ALTER FUNCTION link_type(character varying)
  OWNER TO onix;

/*
  gets a Link_rule by its natural key.
  use: select * from link_rule('the_link_rule_key')
 */
CREATE OR REPLACE FUNCTION link_rule(key_param character varying)
  RETURNS TABLE(
    id bigint,
    key character varying(300),
    name character varying(200),
    description text,
    link_type_key character varying,
    start_item_type_key character varying,
    end_item_type_key character varying,
    version bigint,
    created timestamp(6) with time zone,
    updated timestamp(6) with time zone,
    changedby character varying(100)
  )
  LANGUAGE 'plpgsql'
  COST 100
  STABLE
AS $BODY$
BEGIN
  RETURN QUERY SELECT
    r.id,
    r.key,
    r.name,
    r.description,
    lt.key AS link_type_key,
    start_item_type.key AS start_item_key,
    end_item_type.key AS end_item_type_key,
    r.version,
    r.created,
    r.updated,
    r.changedby
  FROM link_rule r
    INNER JOIN item_type start_item_type
      ON r.start_item_type_id = start_item_type.id
    INNER JOIN item_type end_item_type
      ON r.end_item_type_id = end_item_type.id
    INNER JOIN link_type lt
      ON r.link_type_id = lt.id
  WHERE r.key = key_param;
END;
$BODY$;

ALTER FUNCTION link_rule(character varying)
  OWNER TO onix;

END
$$;